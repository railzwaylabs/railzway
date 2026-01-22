package domain

import (
	"context"
	"errors"

	"github.com/bwmarrin/snowflake"
)

var (
	ErrQuotaDisabled                = errors.New("quota_disabled")
	ErrOrgCustomerQuotaExceeded     = errors.New("org_customer_quota_exceeded")
	ErrOrgSubscriptionQuotaExceeded = errors.New("org_subscription_quota_exceeded")
	ErrOrgUsageQuotaExceeded        = errors.New("org_usage_quota_exceeded")
)

type Service interface {
	// Check if action is allowed
	CanCreateCustomer(ctx context.Context, orgID snowflake.ID) error
	CanCreateSubscription(ctx context.Context, orgID snowflake.ID) error
	CanIngestUsage(ctx context.Context, orgID snowflake.ID) error

	// Get current usage
	GetOrgUsage(ctx context.Context, orgID snowflake.ID) (map[string]int64, error)
}
