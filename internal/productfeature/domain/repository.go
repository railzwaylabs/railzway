package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	ListByProduct(ctx context.Context, orgID, productID uuid.UUID) ([]FeatureAssignment, error)
	ListByProducts(ctx context.Context, orgID uuid.UUID, productIDs []uuid.UUID) ([]FeatureAssignment, error)
	Replace(ctx context.Context, productID uuid.UUID, featureIDs []uuid.UUID, now time.Time) error
}
