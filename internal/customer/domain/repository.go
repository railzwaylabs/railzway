package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListFilter struct {
	Name        string
	Email       string
	Currency    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	Create(ctx context.Context, customer Customer) error
	Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	FindByID(ctx context.Context, orgID, id uuid.UUID) (*Customer, error)
	FindByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Customer, error)
	List(ctx context.Context, orgID uuid.UUID, filter ListFilter, limit int, cursor *ListCursor) ([]*Customer, error)
	Count(ctx context.Context, orgID uuid.UUID) (int64, error)
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}
