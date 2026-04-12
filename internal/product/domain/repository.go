package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListFilter struct {
	Code   string
	Name   string
	Active *bool
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	Create(ctx context.Context, product Product) error
	Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	FindByID(ctx context.Context, orgID, id uuid.UUID) (*Product, error)
	FindByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Product, error)
	List(ctx context.Context, orgID uuid.UUID, filter ListFilter, limit int, cursor *ListCursor) ([]*Product, error)
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) ([]*Product, error)
}
