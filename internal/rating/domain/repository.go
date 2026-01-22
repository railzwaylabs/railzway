package domain

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	subscriptiondomain "github.com/smallbiznis/railzway/internal/subscription/domain"
)

type Repository interface {
	GetBillingCycle(ctx context.Context, id snowflake.ID) (*BillingCycleRow, error)
	GetSubscription(ctx context.Context, orgID, subID snowflake.ID) (*subscriptiondomain.Subscription, error)
	ListSubscriptionItems(ctx context.Context, orgID, subID snowflake.ID) ([]SubscriptionItemRow, error)
	ListEntitlements(ctx context.Context, orgID, subID snowflake.ID, start, end time.Time) ([]subscriptiondomain.SubscriptionEntitlement, error)
	AggregateUsage(ctx context.Context, orgID, subID, meterID snowflake.ID, start, end time.Time) (float64, error)
	DeleteRatingResults(ctx context.Context, cycleID snowflake.ID) error
	InsertRatingResult(ctx context.Context, result RatingResult) error
}
