package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

const (
	StatusTrialing = "trialing"
	StatusActive   = "active"
	StatusPastDue  = "past_due"
	StatusCanceled = "canceled"
	StatusPaused   = "paused"

	PeriodStatusOpen   = "open"
	PeriodStatusClosed = "closed"
)

type CreateSubscriptionRequest struct {
	CustomerID         string
	PlanID             string
	Currency           string
	StartAt            *time.Time
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	TrialEnd           *time.Time
	CancelAt           *time.Time
	Status             string
	IdempotencyKey     string
	Items              []CreateSubscriptionItemInput
}

type UpdateSubscriptionRequest struct {
	Status     *string
	CancelAt   *time.Time
	CanceledAt *time.Time
	EndedAt    *time.Time
}

type GetSubscriptionRequest struct {
	ID string
}

type ListSubscriptionRequest struct {
	PageToken  string
	PageSize   int32
	CustomerID string
	Status     string
}

type SubscriptionResponse struct {
	ID                 string          `json:"id"`
	OrgID              string          `json:"org_id"`
	CustomerID         string          `json:"customer_id"`
	PlanID             string          `json:"plan_id"`
	Status             string          `json:"status"`
	Currency           string          `json:"currency"`
	StartAt            time.Time       `json:"start_at"`
	CurrentPeriodStart time.Time       `json:"current_period_start"`
	CurrentPeriodEnd   time.Time       `json:"current_period_end"`
	TrialEnd           *time.Time      `json:"trial_end,omitempty"`
	CancelAt           *time.Time      `json:"cancel_at,omitempty"`
	CanceledAt         *time.Time      `json:"canceled_at,omitempty"`
	EndedAt            *time.Time      `json:"ended_at,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Items              []SubscriptionItemResponse `json:"items,omitempty"`
}

type ListSubscriptionResponse struct {
	pagination.PageInfo
	Subscriptions []SubscriptionResponse `json:"subscriptions"`
}

type CreateSubscriptionItemRequest struct {
	SubscriptionID string
	PlanPriceID    string
	Quantity       float64
	StartAt        *time.Time
	EndAt          *time.Time
	IdempotencyKey string
}

type CreateSubscriptionItemInput struct {
	PlanPriceID    string
	Quantity       float64
	StartAt        *time.Time
	EndAt          *time.Time
	IdempotencyKey string
}

type GetSubscriptionItemRequest struct {
	ID string
}

type ListSubscriptionItemRequest struct {
	PageToken      string
	PageSize       int32
	SubscriptionID string
	PlanPriceID    string
}

type SubscriptionItemResponse struct {
	ID             string          `json:"id"`
	OrgID          string          `json:"org_id"`
	SubscriptionID string          `json:"subscription_id"`
	PlanPriceID    string          `json:"plan_price_id"`
	Quantity       float64         `json:"quantity"`
	StartAt        time.Time       `json:"start_at"`
	EndAt          *time.Time      `json:"end_at,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ListSubscriptionItemResponse struct {
	pagination.PageInfo
	Items []SubscriptionItemResponse `json:"items"`
}

type Service interface {
	CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (SubscriptionResponse, error)
	UpdateSubscription(ctx context.Context, id string, req UpdateSubscriptionRequest) (SubscriptionResponse, error)
	GetSubscriptionByID(ctx context.Context, req GetSubscriptionRequest) (SubscriptionResponse, error)
	ListSubscriptions(ctx context.Context, req ListSubscriptionRequest) (ListSubscriptionResponse, error)

	CreateSubscriptionItem(ctx context.Context, req CreateSubscriptionItemRequest) (SubscriptionItemResponse, error)
	GetSubscriptionItemByID(ctx context.Context, req GetSubscriptionItemRequest) (SubscriptionItemResponse, error)
	ListSubscriptionItems(ctx context.Context, req ListSubscriptionItemRequest) (ListSubscriptionItemResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidCustomer     = errors.New("invalid_customer")
	ErrInvalidPlan         = errors.New("invalid_plan")
	ErrInvalidPlanPrice    = errors.New("invalid_plan_price")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrInvalidCurrency     = errors.New("invalid_currency")
	ErrInvalidPeriod       = errors.New("invalid_period")
	ErrInvalidQuantity     = errors.New("invalid_quantity")
	ErrMissingItems        = errors.New("missing_items")
	ErrNotFound            = errors.New("not_found")
)
