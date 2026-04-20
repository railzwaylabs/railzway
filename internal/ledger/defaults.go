package ledger

import (
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/ledger/domain"
)

// DefaultAccounts returns a baseline chart of accounts for a new organization.
func DefaultAccounts(orgID uuid.UUID) []domain.LedgerAccount {
	now := time.Now().UTC()
	return []domain.LedgerAccount{
		{ID: uuid.New(), OrgID: orgID, Code: "1000_cash", Type: domain.LedgerAccountTypeAssets, Name: "Cash", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "1100_accounts_receivable", Type: domain.LedgerAccountTypeAssets, Name: "Accounts Receivable", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "2000_deferred_revenue", Type: domain.LedgerAccountTypeLiability, Name: "Deferred Revenue", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "2100_tax_payable", Type: domain.LedgerAccountTypeLiability, Name: "Tax Payable", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "credits", Type: domain.LedgerAccountTypeLiability, Name: "Customer Credits", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "4000_revenue", Type: domain.LedgerAccountTypeIncome, Name: "Revenue", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "5000_payment_fees", Type: domain.LedgerAccountTypeExpense, Name: "Payment Fees", CreatedAt: now},
		{ID: uuid.New(), OrgID: orgID, Code: "5100_refunds", Type: domain.LedgerAccountTypeExpense, Name: "Refunds", CreatedAt: now},
	}
}
