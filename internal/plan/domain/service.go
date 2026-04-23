package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

const (
	PriceTypeFlat   = "flat"
	PriceTypeUsage  = "usage"
	PriceTypeTiered = "tiered"

	TierModeGraduated = "graduated"
	TierModeVolume    = "volume"

	BillingIntervalDay   = "day"
	BillingIntervalWeek  = "week"
	BillingIntervalMonth = "month"
	BillingIntervalYear  = "year"
)

type CreatePlanRequest struct {
	Code           string
	Name           string
	Description    string
	Active         *bool
	ProductID      *string
	IdempotencyKey string
	Prices         []CreatePlanPriceInput
}

type CreatePlanPriceInput struct {
	Code                 string
	Name                 string
	Description          string
	PriceType            string
	BillingInterval      string
	BillingIntervalCount int
	AggregateUsage       string
	BillingUnit          string
	MeterID              *string
	MeterCode            string
	Active               *bool
	IdempotencyKey       string
	Amounts              []CreatePlanAmountInput
	Tiers                []CreatePlanTierInput
}

type CreatePlanAmountInput struct {
	Currency           string
	UnitAmountCents    float64
	MinimumAmountCents *float64
	MaximumAmountCents *float64
	EffectiveFrom      *time.Time
	EffectiveTo        *time.Time
	IdempotencyKey     string
}

type CreatePlanTierInput struct {
	TierMode        string
	StartQuantity   float64
	EndQuantity     *float64
	UnitAmountCents *float64
	FlatAmountCents *float64
	Unit            string
	IdempotencyKey  string
}

type UpdatePlanRequest struct {
	Name        *string
	Description *string
	Active      *bool
	ProductID   *string
}

type GetPlanRequest struct {
	ID string
}

type ListPlanRequest struct {
	PageToken string
	PageSize  int32
	ProductID *string
	Code      string
	Name      string
	Active    *bool
}

type PlanResponse struct {
	ID          string              `json:"id"`
	OrgID       string              `json:"org_id"`
	ProductID   *string             `json:"product_id,omitempty"`
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Active      bool                `json:"active"`
	Metadata    json.RawMessage     `json:"metadata,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Prices      []PlanPriceResponse `json:"prices,omitempty"`
}

type ListPlanResponse struct {
	pagination.PageInfo
	Plans []PlanResponse `json:"plans"`
}

type CreatePlanPriceRequest struct {
	PlanID               string
	MeterID              *string
	Code                 string
	Name                 string
	Description          string
	PriceType            string
	BillingInterval      string
	BillingIntervalCount int
	AggregateUsage       string
	BillingUnit          string
	MeterCode            string
	Active               *bool
	IdempotencyKey       string
}

type GetPlanPriceRequest struct {
	ID string
}

type ListPlanPriceRequest struct {
	PageToken       string
	PageSize        int32
	PlanID          string
	PriceType       string
	Active          *bool
	BillingInterval string
}

type PlanPriceResponse struct {
	ID                   string               `json:"id"`
	OrgID                string               `json:"org_id"`
	PlanID               string               `json:"plan_id"`
	MeterID              *string              `json:"meter_id,omitempty"`
	Code                 string               `json:"code"`
	Name                 string               `json:"name,omitempty"`
	Description          string               `json:"description,omitempty"`
	PriceType            string               `json:"price_type"`
	BillingInterval      string               `json:"billing_interval"`
	BillingIntervalCount int                  `json:"billing_interval_count"`
	AggregateUsage       string               `json:"aggregate_usage,omitempty"`
	BillingUnit          string               `json:"billing_unit,omitempty"`
	MeterCode            string               `json:"meter_code,omitempty"`
	Active               bool                 `json:"active"`
	Metadata             json.RawMessage      `json:"metadata,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	Amounts              []PlanAmountResponse `json:"amounts,omitempty"`
	Tiers                []PlanTierResponse   `json:"tiers,omitempty"`
}

type ListPlanPriceResponse struct {
	pagination.PageInfo
	Prices []PlanPriceResponse `json:"prices"`
}

type CreatePlanAmountRequest struct {
	PlanPriceID        string
	Currency           string
	UnitAmountCents    float64
	MinimumAmountCents *float64
	MaximumAmountCents *float64
	EffectiveFrom      *time.Time
	EffectiveTo        *time.Time
	IdempotencyKey     string
}

type GetPlanAmountRequest struct {
	ID string
}

type ListPlanAmountRequest struct {
	PageToken   string
	PageSize    int32
	PlanPriceID string
	Currency    string
}

type PlanAmountResponse struct {
	ID                 string          `json:"id"`
	OrgID              string          `json:"org_id"`
	PlanPriceID        string          `json:"plan_price_id"`
	Currency           string          `json:"currency"`
	UnitAmountCents    float64         `json:"unit_amount_cents"`
	MinimumAmountCents *float64        `json:"minimum_amount_cents,omitempty"`
	MaximumAmountCents *float64        `json:"maximum_amount_cents,omitempty"`
	EffectiveFrom      time.Time       `json:"effective_from"`
	EffectiveTo        *time.Time      `json:"effective_to,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type ListPlanAmountResponse struct {
	pagination.PageInfo
	Amounts []PlanAmountResponse `json:"amounts"`
}

type CreatePlanTierRequest struct {
	PlanPriceID     string
	TierMode        string
	StartQuantity   float64
	EndQuantity     *float64
	UnitAmountCents *float64
	FlatAmountCents *float64
	Unit            string
	IdempotencyKey  string
}

type GetPlanTierRequest struct {
	ID string
}

type ListPlanTierRequest struct {
	PageToken   string
	PageSize    int32
	PlanPriceID string
	TierMode    string
}

type PlanTierResponse struct {
	ID              string          `json:"id"`
	OrgID           string          `json:"org_id"`
	PlanPriceID     string          `json:"plan_price_id"`
	TierMode        string          `json:"tier_mode"`
	StartQuantity   float64         `json:"start_quantity"`
	EndQuantity     *float64        `json:"end_quantity,omitempty"`
	UnitAmountCents *float64        `json:"unit_amount_cents,omitempty"`
	FlatAmountCents *float64        `json:"flat_amount_cents,omitempty"`
	Unit            string          `json:"unit"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ListPlanTierResponse struct {
	pagination.PageInfo
	Tiers []PlanTierResponse `json:"tiers"`
}

type Service interface {
	CreatePlan(ctx context.Context, req CreatePlanRequest) (PlanResponse, error)
	UpdatePlan(ctx context.Context, id string, req UpdatePlanRequest) (PlanResponse, error)
	GetPlanByID(ctx context.Context, req GetPlanRequest) (PlanResponse, error)
	ListPlans(ctx context.Context, req ListPlanRequest) (ListPlanResponse, error)

	CreatePlanPrice(ctx context.Context, req CreatePlanPriceRequest) (PlanPriceResponse, error)
	GetPlanPriceByID(ctx context.Context, req GetPlanPriceRequest) (PlanPriceResponse, error)
	ListPlanPrices(ctx context.Context, req ListPlanPriceRequest) (ListPlanPriceResponse, error)

	CreatePlanAmount(ctx context.Context, req CreatePlanAmountRequest) (PlanAmountResponse, error)
	GetPlanAmountByID(ctx context.Context, req GetPlanAmountRequest) (PlanAmountResponse, error)
	ListPlanAmounts(ctx context.Context, req ListPlanAmountRequest) (ListPlanAmountResponse, error)

	CreatePlanTier(ctx context.Context, req CreatePlanTierRequest) (PlanTierResponse, error)
	GetPlanTierByID(ctx context.Context, req GetPlanTierRequest) (PlanTierResponse, error)
	ListPlanTiers(ctx context.Context, req ListPlanTierRequest) (ListPlanTierResponse, error)
}

var (
	ErrInvalidOrganization  = errors.New("invalid_organization")
	ErrInvalidID            = errors.New("invalid_id")
	ErrInvalidCode          = errors.New("invalid_code")
	ErrInvalidName          = errors.New("invalid_name")
	ErrInvalidPriceType     = errors.New("invalid_price_type")
	ErrInvalidInterval      = errors.New("invalid_billing_interval")
	ErrInvalidIntervalCount = errors.New("invalid_billing_interval_count")
	ErrInvalidCurrency      = errors.New("invalid_currency")
	ErrInvalidTierMode      = errors.New("invalid_tier_mode")
	ErrInvalidQuantity      = errors.New("invalid_quantity")
	ErrInvalidAmount        = errors.New("invalid_amount")
	ErrInvalidMeter         = errors.New("invalid_meter")
	ErrInvalidProduct       = errors.New("invalid_product")
	ErrNotFound             = errors.New("not_found")
)
