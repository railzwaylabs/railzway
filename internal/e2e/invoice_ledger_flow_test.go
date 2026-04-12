//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	invoicerepo "github.com/railzwaylabs/railzway/internal/invoice/repository"
	invoiceservice "github.com/railzwaylabs/railzway/internal/invoice/service"
	ledgerdomain "github.com/railzwaylabs/railzway/internal/ledger/domain"
	ledgerrepo "github.com/railzwaylabs/railzway/internal/ledger/repository"
	ledgerservice "github.com/railzwaylabs/railzway/internal/ledger/service"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	"gorm.io/gorm"
)

func TestInvoiceLedgerPosting_OpenInvoiceBalanced(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	orgID, subID := seedFlatSubscription(t, tx, now, periodStart, periodEnd, periodStart, nil)
	seedLedgerAccounts(t, tx, orgID)

	ledgerSvc := ledgerservice.NewService(ledgerservice.Params{
		DB:   tx,
		Repo: ledgerrepo.NewRepository(tx),
	})
	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoicerepo.NewRepository(tx),
		PlanRepo: planrepo.NewRepository(tx),
		SubRepo:  subrepo.NewRepository(tx),
		Ledger:   ledgerSvc,
	})

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-ledger-open",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	openResp, err := invoiceSvc.OpenInvoice(ctx, invoicedomain.OpenInvoiceRequest{ID: resp.Invoice.ID.String()})
	if err != nil {
		t.Fatalf("open invoice: %v", err)
	}

	type sums struct {
		Debit  int64
		Credit int64
	}
	var s sums
	if err := tx.Raw(
		`SELECT
			COALESCE(SUM(CASE WHEN e.entry_type = 'debit' THEN e.amount_cents ELSE 0 END), 0) AS debit,
			COALESCE(SUM(CASE WHEN e.entry_type = 'credit' THEN e.amount_cents ELSE 0 END), 0) AS credit
		 FROM ledger_entries e
		 JOIN ledger_transactions t ON t.id = e.transaction_id
		 WHERE t.org_id = ? AND t.invoice_id = ?`,
		orgID, openResp.Invoice.ID,
	).Scan(&s).Error; err != nil {
		t.Fatalf("ledger sums: %v", err)
	}
	if s.Debit != s.Credit {
		t.Fatalf("ledger not balanced: debit %d credit %d", s.Debit, s.Credit)
	}
	if s.Debit != openResp.Invoice.TotalCents {
		t.Fatalf("expected ledger total %d, got %d", openResp.Invoice.TotalCents, s.Debit)
	}
}

func seedLedgerAccounts(t *testing.T, tx *gorm.DB, orgID uuid.UUID) {
	t.Helper()

	now := time.Now().UTC()
	accounts := []ledgerdomain.LedgerAccount{
		{
			ID:        uuid.New(),
			OrgID:     orgID,
			Code:      "1100_accounts_receivable",
			Type:      ledgerdomain.LedgerAccountTypeAssets,
			Name:      "Accounts Receivable",
			CreatedAt: now,
		},
		{
			ID:        uuid.New(),
			OrgID:     orgID,
			Code:      "4000_revenue",
			Type:      ledgerdomain.LedgerAccountTypeIncome,
			Name:      "Revenue",
			CreatedAt: now,
		},
		{
			ID:        uuid.New(),
			OrgID:     orgID,
			Code:      "2100_tax_payable",
			Type:      ledgerdomain.LedgerAccountTypeLiability,
			Name:      "Tax Payable",
			CreatedAt: now,
		},
	}

	for _, account := range accounts {
		if err := tx.Create(&account).Error; err != nil {
			t.Fatalf("create ledger account: %v", err)
		}
	}

}
