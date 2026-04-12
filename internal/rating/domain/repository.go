package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateRatingResult(ctx context.Context, result RatingResult) error
	FindRatingByUsageEvent(ctx context.Context, orgID, usageEventID uuid.UUID) (*RatingResult, error)
	ListRatingResults(ctx context.Context, orgID uuid.UUID, filter RatingResultFilter, limit int, cursor *ListCursor) ([]*RatingResult, error)

	UpsertUsageAggregate(ctx context.Context, aggregate UsageAggregate) error
	CreateUsageAggregate(ctx context.Context, aggregate UsageAggregate) error
	UpdateUsageAggregate(ctx context.Context, aggregate UsageAggregate) error
	GetUsageAggregate(ctx context.Context, orgID, customerID, planPriceID, planAmountID, meterID uuid.UUID, periodStart, periodEnd time.Time) (*UsageAggregate, error)
	GetUsageAggregateForUpdate(ctx context.Context, orgID, customerID, planPriceID, planAmountID, meterID uuid.UUID, periodStart, periodEnd time.Time) (*UsageAggregate, error)
	ListUsageAggregates(ctx context.Context, orgID uuid.UUID, filter UsageAggregateFilter, limit int, cursor *ListCursor) ([]*UsageAggregate, error)
}

type ListCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type RatingResultFilter struct {
	CustomerID      uuid.UUID
	SubscriptionID  uuid.UUID
	PlanPriceID     uuid.UUID
	MeterID         uuid.UUID
	UsageEventID    uuid.UUID
	WindowStartFrom *time.Time
	WindowStartTo   *time.Time
}

type UsageAggregateFilter struct {
	CustomerID      uuid.UUID
	SubscriptionID  uuid.UUID
	PlanPriceID     uuid.UUID
	MeterID         uuid.UUID
	PeriodStartFrom *time.Time
	PeriodStartTo   *time.Time
}
