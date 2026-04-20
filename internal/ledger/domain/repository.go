package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateAccount(ctx context.Context, account LedgerAccount) error
	FindAccountByCode(ctx context.Context, orgID uuid.UUID, code string) (*LedgerAccount, error)
	ListAccounts(ctx context.Context, orgID uuid.UUID) ([]LedgerAccount, error)

	CreateTransaction(ctx context.Context, tx LedgerTransaction) error
	FindTransactionByID(ctx context.Context, orgID, txID uuid.UUID) (*LedgerTransaction, error)
	FindTransactionByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*LedgerTransaction, error)
	ListTransactionsBySource(ctx context.Context, orgID uuid.UUID, sourceType string, sourceID uuid.UUID, limit int) ([]LedgerTransaction, error)
	ListTransactions(ctx context.Context, orgID uuid.UUID, filter TransactionListFilter, limit int, cursor *ListCursor) ([]LedgerTransaction, error)

	CreateEntries(ctx context.Context, entries []LedgerEntry) error
	ListEntriesByTransaction(ctx context.Context, orgID, transactionID uuid.UUID) ([]LedgerEntry, error)
	GetBalance(ctx context.Context, orgID uuid.UUID, customerID uuid.UUID, accountCode string, currency string) (int64, error)
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type TransactionListFilter struct {
	SourceType     string
	SourceID       uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID uuid.UUID
	InvoiceID      uuid.UUID
	PlanPriceID    uuid.UUID
	MeterID        uuid.UUID
	OccurredFrom   *time.Time
	OccurredTo     *time.Time
	PostedFrom     *time.Time
	PostedTo       *time.Time
}
