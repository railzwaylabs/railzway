package domain

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	GetByOrgID(ctx context.Context, orgID uuid.UUID) (*TestClock, error)
	Upsert(ctx context.Context, clock TestClock) error
	Update(ctx context.Context, orgID uuid.UUID, updates map[string]interface{}) error
	ListActive(ctx context.Context) ([]TestClock, error)
}
