package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	ListByPlan(ctx context.Context, orgID, planID uuid.UUID) ([]FeatureAssignment, error)
	ListByPlans(ctx context.Context, orgID uuid.UUID, planIDs []uuid.UUID) ([]FeatureAssignment, error)
	Replace(ctx context.Context, orgID, planID uuid.UUID, items []PlanFeature, now time.Time) error
}
