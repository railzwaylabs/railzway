package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	ListTaxRates(ctx context.Context, orgID uuid.UUID, filter TaxRateListFilter, limit int, cursor *ListCursor) ([]*TaxRate, error)
	CreateTaxRate(ctx context.Context, rate *TaxRate) error
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type TaxRateListFilter struct {
	Code        string
	Name        string
	Active      *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}
