package domain

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Response, error)
	List(ctx context.Context, opts ListOptions) (ListResponse, error)
	Get(ctx context.Context, id string) (*Response, error)
}

type ListResponse struct {
	PageInfo pagination.PageInfo `json:"page_info"`
	Prices   []Response          `json:"prices"`
}

type CreateRequest struct {
	ProductID            string          `json:"product_id"`
	Code                 string          `json:"code"`
	LookupKey            string          `json:"lookup_key"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	PricingModel         PricingModel    `json:"pricing_model"`
	BillingMode          BillingMode     `json:"billing_mode"`
	BillingInterval      BillingInterval `json:"billing_interval"`
	BillingIntervalCount int32           `json:"billing_interval_count"`
	AggregateUsage       *AggregateUsage `json:"aggregate_usage"`
	BillingUnit          *BillingUnit    `json:"billing_unit"`
	BillingThreshold     *float64        `json:"billing_threshold"`
	TaxBehavior          TaxBehavior     `json:"tax_behavior"`
	TaxCode              *string         `json:"tax_code"`
	Version              *int32          `json:"version"`
	IsDefault            *bool           `json:"is_default"`
	Active               *bool           `json:"active"`
	RetiredAt            *time.Time      `json:"retired_at"`
	Metadata             map[string]any  `json:"metadata"`
	IdempotencyKey       string          `json:"-"`
}

type Response struct {
	ID                   snowflake.ID    `json:"id"`
	OrganizationID       snowflake.ID    `json:"organization_id"`
	ProductID            snowflake.ID    `json:"product_id"`
	Code                 string          `json:"code"`
	LookupKey            *string         `json:"lookup_key,omitempty"`
	Name                 string          `json:"name,omitempty"`
	Description          string          `json:"description,omitempty"`
	PricingModel         PricingModel    `json:"pricing_model"`
	BillingMode          BillingMode     `json:"billing_mode"`
	BillingInterval      BillingInterval `json:"billing_interval"`
	BillingIntervalCount int32           `json:"billing_interval_count"`
	AggregateUsage       *AggregateUsage `json:"aggregate_usage,omitempty"`
	BillingUnit          *BillingUnit    `json:"billing_unit,omitempty"`
	BillingThreshold     *float64        `json:"billing_threshold,omitempty"`
	TaxBehavior          TaxBehavior     `json:"tax_behavior"`
	TaxCode              *string         `json:"tax_code,omitempty"`
	Version              int32           `json:"version"`
	IsDefault            bool            `json:"is_default"`
	Active               bool            `json:"active"`
	RetiredAt            *time.Time      `json:"retired_at,omitempty"`
	Metadata             map[string]any  `json:"metadata"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

var (
	ErrInvalidOrganization         = errors.New("invalid_organization")
	ErrInvalidProduct              = errors.New("invalid_product")
	ErrInvalidCode                 = errors.New("invalid_code")
	ErrInvalidPricingModel         = errors.New("invalid_pricing_model")
	ErrInvalidBillingMode          = errors.New("invalid_billing_mode")
	ErrInvalidBillingInterval      = errors.New("invalid_billing_interval")
	ErrInvalidBillingIntervalCount = errors.New("invalid_billing_interval_count")
	ErrUnsupportedPricingModel     = errors.New("unsupported_pricing_model")
	ErrInvalidAggregateUsage       = errors.New("invalid_aggregate_usage")
	ErrInvalidBillingUnit          = errors.New("invalid_billing_unit")
	ErrInvalidBillingThreshold     = errors.New("invalid_billing_threshold")
	ErrInvalidTaxBehavior          = errors.New("invalid_tax_behavior")
	ErrInvalidVersion              = errors.New("invalid_version")
	ErrInvalidID                   = errors.New("invalid_id")
	ErrNotFound                    = errors.New("not_found")
)
