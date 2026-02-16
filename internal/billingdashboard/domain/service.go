package domain

import (
	"context"
	"errors"
)

// Service exposes admin billing dashboard data.
type Service interface {
	ListCustomerBalances(ctx context.Context, pageToken string, pageSize int32) (CustomerBalancesResponse, error)
	ListBillingCycles(ctx context.Context, pageToken string, pageSize int32) (BillingCycleSummaryResponse, error)
	ListBillingActivity(ctx context.Context, pageToken string, pageSize int32) (BillingActivityResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
)
