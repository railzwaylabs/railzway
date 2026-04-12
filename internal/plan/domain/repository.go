package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type PlanListFilter struct {
	ProductID *uuid.UUID
	Code      string
	Name      string
	Active    *bool
}

type PlanPriceListFilter struct {
	PlanID          uuid.UUID
	PriceType       string
	Active          *bool
	BillingInterval string
}

type PlanAmountListFilter struct {
	PlanPriceID uuid.UUID
	Currency    string
}

type PlanTierListFilter struct {
	PlanPriceID uuid.UUID
	TierMode    string
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreatePlan(ctx context.Context, plan Plan) error
	UpdatePlan(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	FindPlanByID(ctx context.Context, orgID, id uuid.UUID) (*Plan, error)
	FindPlanByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Plan, error)
	ListPlans(ctx context.Context, orgID uuid.UUID, filter PlanListFilter, limit int, cursor *ListCursor) ([]*Plan, error)

	CreatePlanPrice(ctx context.Context, price PlanPrice) error
	FindPlanPriceByID(ctx context.Context, orgID, id uuid.UUID) (*PlanPrice, error)
	FindPlanPriceByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*PlanPrice, error)
	ListPlanPrices(ctx context.Context, orgID uuid.UUID, filter PlanPriceListFilter, limit int, cursor *ListCursor) ([]*PlanPrice, error)

	CreatePlanAmount(ctx context.Context, amount PlanAmount) error
	FindPlanAmountByID(ctx context.Context, orgID, id uuid.UUID) (*PlanAmount, error)
	FindPlanAmountByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*PlanAmount, error)
	ListPlanAmounts(ctx context.Context, orgID uuid.UUID, filter PlanAmountListFilter, limit int, cursor *ListCursor) ([]*PlanAmount, error)

	CreatePlanTier(ctx context.Context, tier PlanTier) error
	FindPlanTierByID(ctx context.Context, orgID, id uuid.UUID) (*PlanTier, error)
	FindPlanTierByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*PlanTier, error)
	ListPlanTiers(ctx context.Context, orgID uuid.UUID, filter PlanTierListFilter, limit int, cursor *ListCursor) ([]*PlanTier, error)
	ListPlanTiersByPrice(ctx context.Context, orgID, planPriceID uuid.UUID) ([]*PlanTier, error)
}
