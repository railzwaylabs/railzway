package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

type CreateProductRequest struct {
	Code           string
	Name           string
	Description    *string
	Active         *bool
	IdempotencyKey string

	FeatureIDs []string
	Plans      []CreateProductPlanInput
}

type CreateProductPlanInput struct {
	Code        string
	Name        string
	Description *string
	Active      *bool

	Prices []CreateProductPlanPriceInput
}

type CreateProductPlanPriceInput struct {
	Code                 string
	Name                 string
	Description          *string
	PriceType            string
	BillingInterval      string
	BillingIntervalCount int
	AggregateUsage       *string
	BillingUnit          *string
	MeterID              *string
	MeterCode            *string
	Active               *bool

	Amounts []CreateProductPlanAmountInput
	Tiers   []CreateProductPlanTierInput
}

type CreateProductPlanAmountInput struct {
	Currency           string
	UnitAmountCents    float64
	MinimumAmountCents *float64
	MaximumAmountCents *float64
	EffectiveFrom      *time.Time
	EffectiveTo        *time.Time
}

type CreateProductPlanTierInput struct {
	TierMode        string
	StartQuantity   float64
	EndQuantity     *float64
	UnitAmountCents *float64
	FlatAmountCents *float64
	Unit            string
}

type UpdateProductRequest struct {
	Name        *string
	Description *string
	Active      *bool
	FeatureIDs  *[]string
}

type GetProductRequest struct {
	ID             string
	ExpandPlans    bool
	ExpandFeatures bool
}

type ListProductRequest struct {
	PageToken      string
	PageSize       int32
	Code           string
	Name           string
	Active         *bool
	ExpandPlans    bool
	ExpandFeatures bool
}

type ProductResponse struct {
	ID          string          `json:"id"`
	OrgID       string          `json:"org_id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Active      bool            `json:"active"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`

	Features []ProductFeatureResponse `json:"features,omitempty"`
	Plans    []ProductPlanResponse    `json:"plans,omitempty"`
}

type ProductFeatureResponse struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	FeatureType string  `json:"feature_type"`
	MeterID     *string `json:"meter_id,omitempty"`
	Active      bool    `json:"active"`
}

type ProductPlanResponse struct {
	ID          string             `json:"id"`
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Active      bool               `json:"active"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Prices      []ProductPlanPrice `json:"prices,omitempty"`
}

type ProductPlanPrice struct {
	ID                   string              `json:"id"`
	Code                 string              `json:"code"`
	Name                 string              `json:"name,omitempty"`
	Description          string              `json:"description,omitempty"`
	PriceType            string              `json:"price_type"`
	BillingInterval      string              `json:"billing_interval"`
	BillingIntervalCount int                 `json:"billing_interval_count"`
	AggregateUsage       string              `json:"aggregate_usage,omitempty"`
	BillingUnit          string              `json:"billing_unit,omitempty"`
	MeterID              *string             `json:"meter_id,omitempty"`
	MeterCode            string              `json:"meter_code,omitempty"`
	Active               bool                `json:"active"`
	Amounts              []ProductPlanAmount `json:"amounts,omitempty"`
	Tiers                []ProductPlanTier   `json:"tiers,omitempty"`
}

type ProductPlanAmount struct {
	ID                 string     `json:"id"`
	Currency           string     `json:"currency"`
	UnitAmountCents    float64    `json:"unit_amount_cents"`
	MinimumAmountCents *float64   `json:"minimum_amount_cents,omitempty"`
	MaximumAmountCents *float64   `json:"maximum_amount_cents,omitempty"`
	EffectiveFrom      time.Time  `json:"effective_from"`
	EffectiveTo        *time.Time `json:"effective_to,omitempty"`
}

type ProductPlanTier struct {
	ID              string   `json:"id"`
	TierMode        string   `json:"tier_mode"`
	StartQuantity   float64  `json:"start_quantity"`
	EndQuantity     *float64 `json:"end_quantity,omitempty"`
	UnitAmountCents *float64 `json:"unit_amount_cents,omitempty"`
	FlatAmountCents *float64 `json:"flat_amount_cents,omitempty"`
	Unit            string   `json:"unit"`
}

type ListProductResponse struct {
	pagination.PageInfo
	Products []ProductResponse `json:"products"`
}

type Service interface {
	Create(ctx context.Context, req CreateProductRequest) (ProductResponse, error)
	Update(ctx context.Context, id string, req UpdateProductRequest) (ProductResponse, error)
	GetByID(ctx context.Context, req GetProductRequest) (ProductResponse, error)
	List(ctx context.Context, req ListProductRequest) (ListProductResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidCode         = errors.New("invalid_code")
	ErrInvalidName         = errors.New("invalid_name")
	ErrNotFound            = errors.New("not_found")
)
