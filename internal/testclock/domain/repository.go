package domain

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*TestClock, error)
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*TestClock, error)
	Create(ctx context.Context, clock TestClock) error
	Upsert(ctx context.Context, clock TestClock) error
	Update(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]TestClock, error)
	ListActive(ctx context.Context) ([]TestClock, error)
}
