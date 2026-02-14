package domain

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

type ListSubscriptionRequest struct {
	Status      string
	CustomerID  string
	PageToken   string
	PageSize    int32
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type ListSubscriptionResponse struct {
	pagination.PageInfo
	Subscriptions []Subscription `json:"subscriptions"`
}

type ListEntitlementsRequest struct {
	SubscriptionID string
	EffectiveAt    *time.Time
	PageToken      string
	PageSize       int32
}

type EntitlementResponse struct {
	ID             snowflake.ID  `json:"id"`
	SubscriptionID snowflake.ID  `json:"subscription_id"`
	ProductID      snowflake.ID  `json:"product_id"`
	FeatureCode    string        `json:"feature_code"`
	FeatureName    string        `json:"feature_name"`
	FeatureType    string        `json:"feature_type"`
	MeterID        *snowflake.ID `json:"meter_id,omitempty"`
	EffectiveFrom  time.Time     `json:"effective_from"`
	EffectiveTo    *time.Time    `json:"effective_to,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

type ListEntitlementsResponse struct {
	pagination.PageInfo
	Entitlements []EntitlementResponse `json:"entitlements"`
}

type CreateSubscriptionItemRequest struct {
	PriceID  string `json:"price_id"`
	Quantity int8   `json:"quantity,omitempty"`
}

type CreateSubscriptionRequest struct {
	CustomerID       string                          `json:"customer_id"`
	CollectionMode   SubscriptionCollectionMode      `json:"collection_mode"`
	BillingCycleType string                          `json:"billing_cycle_type"`
	Items            []CreateSubscriptionItemRequest `json:"items"`
	TrialDays        *int                            `json:"trial_days,omitempty"`
	Metadata         map[string]any                  `json:"metadata,omitempty"`
	IdempotencyKey   string                          `json:"-"`
}

type ReplaceSubscriptionItemsRequest struct {
	SubscriptionID string                          `json:"subscription_id"`
	Items          []CreateSubscriptionItemRequest `json:"items"`
}

type GetActiveByCustomerIDRequest struct {
	CustomerID string
}

type GetSubscriptionItemRequest struct {
	SubscriptionID string
	MeterID        string
	MeterCode      string
}

type TransitionReason string

//go:generate mockgen -source=service.go -destination=./mocks/mock_service.go -package=mocks
type Service interface {
	List(context.Context, ListSubscriptionRequest) (ListSubscriptionResponse, error)
	ListEntitlements(context.Context, ListEntitlementsRequest) (ListEntitlementsResponse, error)
	Create(context.Context, CreateSubscriptionRequest) (CreateSubscriptionResponse, error)
	ReplaceItems(context.Context, ReplaceSubscriptionItemsRequest) (CreateSubscriptionResponse, error)
	GetByID(context.Context, string) (Subscription, error)
	GetActiveByCustomerID(context.Context, GetActiveByCustomerIDRequest) (Subscription, error)
	GetSubscriptionItem(context.Context, GetSubscriptionItemRequest) (SubscriptionItem, error)
	TransitionSubscription(ctx context.Context, subscriptionID string, targetStatus SubscriptionStatus, reason TransitionReason) error
	ValidateUsageEntitlement(ctx context.Context, subscriptionID, meterID snowflake.ID, at time.Time) error
	ChangePlan(ctx context.Context, req ChangePlanRequest) error
}

type ChangePlanRequest struct {
	SubscriptionID string
	NewProductID   string
}

type CreateSubscriptionItemResponse struct {
	ID                string   `json:"id"`
	PriceID           string   `json:"price_id"`
	PriceCode         *string  `json:"price_code,omitempty"`
	MeterID           *string  `json:"meter_id,omitempty"`
	MeterCode         *string  `json:"meter_code,omitempty"`
	Quantity          int8     `json:"quantity"`
	BillingMode       string   `json:"billing_mode"`
	UsageBehavior     *string  `json:"usage_behavior,omitempty"`
	BillingThreshold  *float64 `json:"billing_threshold,omitempty"`
	ProrationBehavior *string  `json:"proration_behavior,omitempty"`
}

type CreateSubscriptionResponse struct {
	ID             string                           `json:"id"`
	OrganizationID string                           `json:"organization_id"`
	CustomerID     string                           `json:"customer_id"`
	Status         SubscriptionStatus               `json:"status"`
	CollectionMode SubscriptionCollectionMode       `json:"collection_mode"`
	StartAt        time.Time                        `json:"start_at"`
	Items          []CreateSubscriptionItemResponse `json:"items"`
	Metadata       map[string]any                   `json:"metadata,omitempty"`
}

var (
	ErrInvalidOrganization       = errors.New("invalid_organization")
	ErrInvalidCustomer           = errors.New("invalid_customer")
	ErrInvalidTrialDays          = errors.New("invalid_trial_days")
	ErrInvalidSubscription       = errors.New("invalid_subscription")
	ErrInvalidMeterID            = errors.New("invalid_meter_id")
	ErrUnsupportedPricingModel   = errors.New("unsupported_pricing_model")
	ErrInvalidMeterCode          = errors.New("invalid_meter_code")
	ErrInvalidStatus             = errors.New("invalid_status")
	ErrInvalidTargetStatus       = errors.New("invalid_target_status")
	ErrInvalidTransition         = errors.New("invalid_transition")
	ErrMissingSubscriptionItems  = errors.New("missing_subscription_items")
	ErrMissingPricing            = errors.New("missing_pricing")
	ErrMissingCustomer           = errors.New("missing_customer")
	ErrMissingEntitlements       = errors.New("missing_entitlements")
	ErrBillingCyclesOpen         = errors.New("billing_cycles_open")
	ErrInvoicesNotFinalized      = errors.New("invoices_not_finalized")
	ErrInvalidCollectionMode     = errors.New("invalid_collection_mode")
	ErrInvalidBillingCycleType   = errors.New("invalid_billing_cycle_type")
	ErrInvalidCurrency           = errors.New("invalid_currency")
	ErrInvalidStartAt            = errors.New("invalid_start_at")
	ErrInvalidPeriod             = errors.New("invalid_period")
	ErrInvalidItems              = errors.New("invalid_items")
	ErrInvalidQuantity           = errors.New("invalid_quantity")
	ErrInvalidPrice              = errors.New("invalid_price")
	ErrInvalidProduct            = errors.New("invalid_product")
	ErrMultipleFlatPrices        = errors.New("multiple_flat_prices_not_allowed")
	ErrSubscriptionNotFound      = errors.New("subscription_not_found")
	ErrSubscriptionItemNotFound  = errors.New("subscription_item_not_found")
	ErrFeatureNotEntitled        = errors.New("feature_not_entitled")
	ErrInvalidSubscriptionStatus = errors.New("invalid_subscription_status")
	ErrMissingPaymentMethod      = errors.New("missing_payment_method")
)
