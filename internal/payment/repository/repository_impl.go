package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
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

func (r *repository) ListPayments(ctx context.Context, orgID uuid.UUID, filter domain.PaymentListFilter, limit int, cursor *domain.ListCursor) ([]*domain.Payment, error) {
	var payments []*domain.Payment
	stmt := r.db.WithContext(ctx).Model(&domain.Payment{}).Where("org_id = ?", orgID)
	if filter.CustomerID != uuid.Nil {
		stmt = stmt.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.InvoiceID != uuid.Nil {
		stmt = stmt.Where("invoice_id = ?", filter.InvoiceID)
	}
	if filter.Status != "" {
		stmt = stmt.Where("status = ?", filter.Status)
	}
	if filter.Provider != "" {
		stmt = stmt.Where("provider = ?", filter.Provider)
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
	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}
