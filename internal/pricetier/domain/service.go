package domain

import (
	"context"
	"errors"
	"time"
)

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Response, error)
	List(ctx context.Context) ([]Response, error)
	Get(ctx context.Context, id string) (*Response, error)
}

type CreateRequest struct {
	PriceID         string         `json:"price_id"`
	TierMode        int16          `json:"tier_mode"`
	StartQuantity   float64        `json:"start_quantity"`
	EndQuantity     *float64       `json:"end_quantity"`
	UnitAmountCents *int64         `json:"unit_amount_cents"`
	FlatAmountCents *int64         `json:"flat_amount_cents"`
	Unit            string         `json:"unit"`
	Metadata        map[string]any `json:"metadata"`
}

type Response struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	PriceID         string    `json:"price_id"`
	TierMode        int16     `json:"tier_mode"`
	StartQuantity   float64   `json:"start_quantity"`
	EndQuantity     *float64  `json:"end_quantity,omitempty"`
	UnitAmountCents *int64    `json:"unit_amount_cents,omitempty"`
	FlatAmountCents *int64    `json:"flat_amount_cents,omitempty"`
	Unit            string    `json:"unit"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidPrice        = errors.New("invalid_price")
	ErrInvalidTierMode     = errors.New("invalid_tier_mode")
	ErrInvalidStartQty     = errors.New("invalid_start_quantity")
	ErrInvalidEndQty       = errors.New("invalid_end_quantity")
	ErrInvalidUnitAmount   = errors.New("invalid_unit_amount")
	ErrInvalidFlatAmount   = errors.New("invalid_flat_amount")
	ErrInvalidUnit         = errors.New("invalid_unit")
	ErrInvalidID           = errors.New("invalid_id")
	ErrNotFound            = errors.New("not_found")
)
