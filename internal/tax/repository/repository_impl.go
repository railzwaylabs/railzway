package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/tax/domain"
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

func (r *repository) ListTaxRates(ctx context.Context, orgID uuid.UUID, filter domain.TaxRateListFilter, limit int, cursor *domain.ListCursor) ([]*domain.TaxRate, error) {
	var rates []*domain.TaxRate
	stmt := r.db.WithContext(ctx).Model(&domain.TaxRate{}).Where("org_id = ?", orgID)
	if filter.Code != "" {
		stmt = stmt.Where("code ILIKE ?", "%"+filter.Code+"%")
	}
	if filter.Name != "" {
		stmt = stmt.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.Active != nil {
		stmt = stmt.Where("active = ?", *filter.Active)
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
	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&rates).Error; err != nil {
		return nil, err
	}
	return rates, nil
}

func (r *repository) CreateTaxRate(ctx context.Context, rate *domain.TaxRate) error {
	return r.db.WithContext(ctx).Create(rate).Error
}
