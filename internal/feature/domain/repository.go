package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListFilter struct {
	Code        string
	Name        string
	FeatureType string
	Active      *bool
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	Create(ctx context.Context, feature Feature) error
	Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	FindByID(ctx context.Context, orgID, id uuid.UUID) (*Feature, error)
	FindByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Feature, error)
	List(ctx context.Context, orgID uuid.UUID, filter ListFilter, limit int, cursor *ListCursor) ([]*Feature, error)
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) ([]*Feature, error)
}
