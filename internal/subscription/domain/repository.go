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

type SubscriptionListFilter struct {
	CustomerID uuid.UUID
	Status     string
}

type SubscriptionItemListFilter struct {
	SubscriptionID uuid.UUID
	PlanPriceID    uuid.UUID
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateSubscription(ctx context.Context, sub Subscription) error
	UpdateSubscription(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
	FindSubscriptionByID(ctx context.Context, orgID, id uuid.UUID) (*Subscription, error)
	FindSubscriptionByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*Subscription, error)
	ListSubscriptions(ctx context.Context, orgID uuid.UUID, filter SubscriptionListFilter, limit int, cursor *ListCursor) ([]*Subscription, error)
	CreateSubscriptionPeriod(ctx context.Context, period SubscriptionPeriod) error
	FindSubscriptionPeriodByTime(ctx context.Context, orgID, subscriptionID uuid.UUID, at time.Time) (*SubscriptionPeriod, error)
	FindSubscriptionPeriodForUpdate(ctx context.Context, orgID, periodID uuid.UUID) (*SubscriptionPeriod, error)
	ListOpenSubscriptionPeriods(ctx context.Context, asOf time.Time, limit int) ([]SubscriptionPeriod, error)
	ListOpenSubscriptionPeriodsByTestClock(ctx context.Context, orgID, testClockID uuid.UUID, asOf time.Time, limit int) ([]SubscriptionPeriod, error)
	UpdateSubscriptionPeriod(ctx context.Context, orgID, periodID uuid.UUID, updates map[string]interface{}) error

	CreateSubscriptionItem(ctx context.Context, item SubscriptionItem) error
	FindSubscriptionItemByID(ctx context.Context, orgID, id uuid.UUID) (*SubscriptionItem, error)
	FindSubscriptionItemByIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*SubscriptionItem, error)
	ListSubscriptionItems(ctx context.Context, orgID uuid.UUID, filter SubscriptionItemListFilter, limit int, cursor *ListCursor) ([]*SubscriptionItem, error)
}
