package domain

import (
	"context"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

const (
	SourceUsage = "usage"
)

type RateUsageRequest struct {
	UsageEventID string
}

type RateUsageResponse struct {
	RatingResult RatingResultResponse `json:"rating_result"`
}

type ListRatingResultsRequest struct {
	PageToken       string
	PageSize        int32
	CustomerID      string
	SubscriptionID  string
	PlanPriceID     string
	MeterID         string
	UsageEventID    string
	WindowStartFrom *time.Time
	WindowStartTo   *time.Time
}

type RatingResultResponse struct {
	ID              string    `json:"id"`
	UsageEventID    string    `json:"usage_event_id"`
	CustomerID      string    `json:"customer_id"`
	SubscriptionID  *string   `json:"subscription_id,omitempty"`
	PlanPriceID     string    `json:"plan_price_id"`
	MeterID         string    `json:"meter_id"`
	Currency        string    `json:"currency"`
	Quantity        float64   `json:"quantity"`
	UnitAmountCents int64     `json:"unit_amount_cents"`
	AmountCents     int64     `json:"amount_cents"`
	Source          string    `json:"source"`
	WindowStart     time.Time `json:"window_start"`
	WindowEnd       time.Time `json:"window_end"`
	CreatedAt       time.Time `json:"created_at"`
}

type ListRatingResultsResponse struct {
	pagination.PageInfo
	Results []RatingResultResponse `json:"results"`
}

type ListUsageAggregatesRequest struct {
	PageToken       string
	PageSize        int32
	CustomerID      string
	SubscriptionID  string
	PlanPriceID     string
	MeterID         string
	PeriodStartFrom *time.Time
	PeriodStartTo   *time.Time
}

type UsageAggregateResponse struct {
	ID             string     `json:"id"`
	OrgID          string     `json:"org_id"`
	CustomerID     string     `json:"customer_id"`
	SubscriptionID *string    `json:"subscription_id,omitempty"`
	PlanPriceID    string     `json:"plan_price_id"`
	PlanAmountID   *string    `json:"plan_amount_id,omitempty"`
	MeterID        string     `json:"meter_id"`
	Currency       string     `json:"currency"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	Quantity       float64    `json:"quantity"`
	AmountCents    int64      `json:"amount_cents"`
	LastEventAt    *time.Time `json:"last_event_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ListUsageAggregatesResponse struct {
	pagination.PageInfo
	Aggregates []UsageAggregateResponse `json:"aggregates"`
}

type Service interface {
	RateUsage(ctx context.Context, req RateUsageRequest) (RateUsageResponse, error)
	ListRatingResults(ctx context.Context, req ListRatingResultsRequest) (ListRatingResultsResponse, error)
	ListUsageAggregates(ctx context.Context, req ListUsageAggregatesRequest) (ListUsageAggregatesResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidUsageEvent   = errors.New("invalid_usage_event")
	ErrUsageNotFound       = errors.New("usage_not_found")
	ErrPricingNotFound     = errors.New("pricing_not_found")
	ErrInvalidAmount       = errors.New("invalid_amount")
	ErrInvalidCustomer     = errors.New("invalid_customer")
	ErrInvalidSubscription = errors.New("invalid_subscription")
	ErrInvalidPlanPrice    = errors.New("invalid_plan_price")
	ErrInvalidMeter        = errors.New("invalid_meter")
	ErrInvalidPeriod       = errors.New("invalid_period")
	ErrInvalidCursor       = errors.New("invalid_cursor")
)
