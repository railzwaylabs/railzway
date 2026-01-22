package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	invoicedomain "github.com/smallbiznis/railzway/internal/invoice/domain"
	ledgerdomain "github.com/smallbiznis/railzway/internal/ledger/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// postInvoiceToLedger creates double-entry ledger postings for a finalized invoice.
// This method MUST be called ONLY within the FinalizeInvoice transaction.
// It reads ONLY from the finalized invoice snapshot (no live lookups).
//
// Double-entry logic:
//
//	Debit:  Accounts Receivable (asset increases)
//	Credit: Revenue (income increases)
//	Credit: Tax Payable (liability increases, if tax > 0)
//
// Idempotency: The ledger service has ON CONFLICT DO NOTHING, so re-posting
// the same invoice will not create duplicate entries.
func (s *Service) postInvoiceToLedger(ctx context.Context, tx *gorm.DB, invoice *invoicedomain.Invoice) error {
	if invoice == nil {
		return fmt.Errorf("invoice is nil")
	}
	if invoice.Status != invoicedomain.InvoiceStatusFinalized {
		return fmt.Errorf("invoice must be finalized before posting to ledger")
	}
	if invoice.FinalizedAt == nil {
		return fmt.Errorf("finalized_at is required for ledger posting")
	}

	// Load ledger accounts by code
	accounts, err := s.loadLedgerAccounts(ctx, tx, invoice.OrgID, []ledgerdomain.LedgerAccountCode{
		ledgerdomain.AccountCodeAccountsReceivable,
		ledgerdomain.AccountCodeRevenueUsage, // Using usage revenue for all revenue
		ledgerdomain.AccountCodeTaxPayable,
	})
	if err != nil {
		return fmt.Errorf("failed to load ledger accounts: %w", err)
	}

	arAccount, ok := accounts[ledgerdomain.AccountCodeAccountsReceivable]
	if !ok {
		return fmt.Errorf("accounts_receivable account not found for org %s", invoice.OrgID)
	}
	revenueAccount, ok := accounts[ledgerdomain.AccountCodeRevenueUsage]
	if !ok {
		return fmt.Errorf("revenue_usage account not found for org %s", invoice.OrgID)
	}

	// Build ledger entry lines
	lines := []ledgerdomain.LedgerEntryLine{
		{
			AccountID: arAccount.ID,
			Direction: ledgerdomain.LedgerEntryDirectionDebit,
			Currency:  invoice.Currency,
			Amount:    invoice.TotalAmount, // Total AR = Revenue + Tax
		},
		{
			AccountID: revenueAccount.ID,
			Direction: ledgerdomain.LedgerEntryDirectionCredit,
			Currency:  invoice.Currency,
			Amount:    invoice.SubtotalAmount, // Revenue = Subtotal (before tax)
		},
	}

	// Add tax payable if applicable
	if invoice.TaxAmount > 0 {
		taxAccount, ok := accounts[ledgerdomain.AccountCodeTaxPayable]
		if !ok {
			return fmt.Errorf("tax_payable account not found for org %s", invoice.OrgID)
		}
		lines = append(lines, ledgerdomain.LedgerEntryLine{
			AccountID: taxAccount.ID,
			Direction: ledgerdomain.LedgerEntryDirectionCredit,
			Currency:  invoice.Currency,
			Amount:    invoice.TaxAmount,
		})
	}

	// Validate balance before posting
	if err := ledgerdomain.ValidateBalanced(lines); err != nil {
		return fmt.Errorf("ledger entry not balanced: %w", err)
	}

	// Post to ledger
	// Note: CreateEntry is NOT transaction-aware, so we need to use the ledger service
	// that can accept a transaction. For now, we'll call it with context and rely on
	// the service's internal transaction handling with idempotency.
	//
	// IMPORTANT: The ledger service's CreateEntry creates its own transaction,
	// but we're already in a transaction. We need to modify the approach.
	//
	// Since the ledger service creates its own transaction, we need to ensure
	// it uses the same transaction we're in. Let's check if we can pass tx directly.

	// For now, we'll use a workaround: call the ledger service's CreateEntry
	// which will create a nested transaction. PostgreSQL supports this via savepoints.
	// SQLite also supports savepoints.

	// However, the cleaner approach is to directly insert into ledger tables
	// within our existing transaction to maintain atomicity.

	return s.postLedgerEntryDirect(ctx, tx, invoice, lines)
}

// postLedgerEntryDirect posts ledger entries directly within the current transaction.
// This ensures atomicity with invoice finalization.
func (s *Service) postLedgerEntryDirect(ctx context.Context, tx *gorm.DB, invoice *invoicedomain.Invoice, lines []ledgerdomain.LedgerEntryLine) error {
	entryID := s.genID.Generate()
	now := time.Now().UTC()

	// Insert ledger entry header with idempotency
	result := tx.WithContext(ctx).Exec(
		`INSERT INTO ledger_entries (
			id, org_id, source_type, source_id, currency, occurred_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (org_id, source_type, source_id) DO NOTHING`,
		entryID,
		invoice.OrgID,
		string(ledgerdomain.SourceTypeBillingCycle),
		invoice.ID,
		invoice.Currency,
		invoice.FinalizedAt.UTC(),
		now,
	)
	if result.Error != nil {
		return fmt.Errorf("failed to insert ledger entry: %w", result.Error)
	}

	// If no rows affected, entry already exists (idempotency)
	if result.RowsAffected == 0 {
		s.log.Info("ledger entry already exists for invoice",
			zap.String("invoice_id", invoice.ID.String()),
			zap.String("org_id", invoice.OrgID.String()),
		)
		return nil
	}

	// Insert ledger entry lines
	for _, line := range lines {
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO ledger_entry_lines (
				id, ledger_entry_id, account_id, direction, currency, amount, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			s.genID.Generate(),
			entryID,
			line.AccountID,
			string(line.Direction),
			line.Currency,
			line.Amount,
			now,
		).Error; err != nil {
			return fmt.Errorf("failed to insert ledger entry line: %w", err)
		}
	}

	s.log.Info("posted invoice to ledger",
		zap.String("invoice_id", invoice.ID.String()),
		zap.String("ledger_entry_id", entryID.String()),
		zap.Int64("total_amount", invoice.TotalAmount),
	)

	return nil
}

// loadLedgerAccounts loads ledger accounts by code for the given organization.
func (s *Service) loadLedgerAccounts(ctx context.Context, tx *gorm.DB, orgID snowflake.ID, codes []ledgerdomain.LedgerAccountCode) (map[ledgerdomain.LedgerAccountCode]ledgerdomain.LedgerAccount, error) {
	var accounts []ledgerdomain.LedgerAccount
	if err := tx.WithContext(ctx).
		Where("org_id = ? AND code IN ?", orgID, codes).
		Find(&accounts).Error; err != nil {
		return nil, err
	}

	result := make(map[ledgerdomain.LedgerAccountCode]ledgerdomain.LedgerAccount)
	for _, acc := range accounts {
		result[acc.Code] = acc
	}

	return result, nil
}
