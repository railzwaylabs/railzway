package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/invoice/domain"
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

func (r *repository) CreateInvoice(ctx context.Context, inv domain.Invoice) error {
	return r.db.WithContext(ctx).Create(&inv).Error
}

func (r *repository) FindInvoiceByID(ctx context.Context, orgID, invoiceID uuid.UUID) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, invoiceID).
		First(&inv).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *repository) FindInvoiceByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND idempotency_key = ?", orgID, key).
		First(&inv).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *repository) FindInvoiceBySubscriptionPeriod(ctx context.Context, orgID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND subscription_id = ? AND period_start = ? AND period_end = ?", orgID, subscriptionID, periodStart, periodEnd).
		Order("created_at desc").
		First(&inv).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *repository) UpdateInvoice(ctx context.Context, orgID, invoiceID uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Invoice{}).
		Where("org_id = ? AND id = ?", orgID, invoiceID).
		Updates(updates).Error
}

func (r *repository) ListInvoices(ctx context.Context, orgID uuid.UUID, filter domain.InvoiceListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Invoice, error) {
	var invoices []*domain.Invoice
	stmt := r.db.WithContext(ctx).Model(&domain.Invoice{}).Where("org_id = ?", orgID)
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.SubscriptionID != uuid.Nil {
		stmt = stmt.Where("subscription_id = ?", filter.SubscriptionID)
	}
	if filter.Status != "" {
		stmt = stmt.Where("status = ?", filter.Status)
	}
	if filter.Number != "" {
		stmt = stmt.Where("number ILIKE ?", "%"+filter.Number+"%")
	}
	if filter.PeriodStartFrom != nil {
		stmt = stmt.Where("period_start >= ?", *filter.PeriodStartFrom)
	}
	if filter.PeriodStartTo != nil {
		stmt = stmt.Where("period_start <= ?", *filter.PeriodStartTo)
	}
	if filter.IssuedFrom != nil {
		stmt = stmt.Where("issued_at >= ?", *filter.IssuedFrom)
	}
	if filter.IssuedTo != nil {
		stmt = stmt.Where("issued_at <= ?", *filter.IssuedTo)
	}
	if filter.CreatedFrom != nil {
		stmt = stmt.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		stmt = stmt.Where("created_at <= ?", *filter.CreatedTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}
	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *repository) ListInvoiceItemsByInvoice(ctx context.Context, orgID, invoiceID uuid.UUID) ([]*domain.InvoiceItem, error) {
	var items []*domain.InvoiceItem
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND invoice_id = ?", orgID, invoiceID).
		Order("created_at asc").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) CreateInvoiceItems(ctx context.Context, items []domain.InvoiceItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

func (r *repository) DeleteInvoiceItemsByInvoice(ctx context.Context, orgID, invoiceID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("org_id = ? AND invoice_id = ?", orgID, invoiceID).
		Delete(&domain.InvoiceItem{}).Error
}
