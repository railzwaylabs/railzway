package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/ledger/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) WithTx(tx *gorm.DB) domain.Repository {
	return &repository{db: tx}
}

func (r *repository) CreateAccount(ctx context.Context, account domain.LedgerAccount) error {
	return r.db.WithContext(ctx).Create(&account).Error
}

func (r *repository) FindAccountByCode(ctx context.Context, orgID uuid.UUID, code string) (*domain.LedgerAccount, error) {
	var account domain.LedgerAccount
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND code = ?", orgID, code).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *repository) ListAccounts(ctx context.Context, orgID uuid.UUID) ([]domain.LedgerAccount, error) {
	var accounts []domain.LedgerAccount
	if err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Order("code asc").
		Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *repository) CreateTransaction(ctx context.Context, tx domain.LedgerTransaction) error {
	return r.db.WithContext(ctx).Create(&tx).Error
}

func (r *repository) FindTransactionByID(ctx context.Context, orgID, txID uuid.UUID) (*domain.LedgerTransaction, error) {
	var tx domain.LedgerTransaction
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, txID).
		First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *repository) FindTransactionByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.LedgerTransaction, error) {
	var tx domain.LedgerTransaction
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *repository) ListTransactionsBySource(ctx context.Context, orgID uuid.UUID, sourceType string, sourceID uuid.UUID, limit int) ([]domain.LedgerTransaction, error) {
	var txs []domain.LedgerTransaction
	stmt := r.db.WithContext(ctx).
		Where("org_id = ? AND source_type = ? AND source_id = ?", orgID, sourceType, sourceID).
		Order("occurred_at desc, id desc")
	if limit > 0 {
		stmt = stmt.Limit(limit)
	}
	if err := stmt.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *repository) ListTransactions(ctx context.Context, orgID uuid.UUID, filter domain.TransactionListFilter, limit int, cursor *domain.ListCursor) ([]domain.LedgerTransaction, error) {
	var txs []domain.LedgerTransaction
	stmt := r.db.WithContext(ctx).Model(&domain.LedgerTransaction{}).Where("org_id = ?", orgID)
	if filter.SourceType != "" {
		stmt = stmt.Where("source_type = ?", filter.SourceType)
	}
	if filter.SourceID != uuid.Nil {
		stmt = stmt.Where("source_id = ?", filter.SourceID)
	}
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.SubscriptionID != uuid.Nil {
		stmt = stmt.Where("subscription_id = ?", filter.SubscriptionID)
	}
	if filter.InvoiceID != uuid.Nil {
		stmt = stmt.Where("invoice_id = ?", filter.InvoiceID)
	}
	if filter.PlanPriceID != uuid.Nil {
		stmt = stmt.Where("plan_price_id = ?", filter.PlanPriceID)
	}
	if filter.MeterID != uuid.Nil {
		stmt = stmt.Where("meter_id = ?", filter.MeterID)
	}
	if filter.OccurredFrom != nil {
		stmt = stmt.Where("occurred_at >= ?", *filter.OccurredFrom)
	}
	if filter.OccurredTo != nil {
		stmt = stmt.Where("occurred_at <= ?", *filter.OccurredTo)
	}
	if filter.PostedFrom != nil {
		stmt = stmt.Where("posted_at >= ?", *filter.PostedFrom)
	}
	if filter.PostedTo != nil {
		stmt = stmt.Where("posted_at <= ?", *filter.PostedTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *repository) CreateEntries(ctx context.Context, entries []domain.LedgerEntry) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&entries).Error
}

func (r *repository) ListEntriesByTransaction(ctx context.Context, orgID, transactionID uuid.UUID) ([]domain.LedgerEntry, error) {
	var entries []domain.LedgerEntry
	if err := r.db.WithContext(ctx).
		Where("org_id = ? AND transaction_id = ?", orgID, transactionID).
		Order("created_at asc, id asc").
		Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}
