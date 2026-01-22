package domain

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
)

// TaxResolver returns the active tax definition for an invoice context.
type TaxResolver interface {
	ResolveForInvoice(ctx context.Context, orgID, customerID snowflake.ID) (*TaxDefinition, error)
}

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Response, error)
	List(ctx context.Context, req ListRequest) ([]Response, error)
	Update(ctx context.Context, req UpdateRequest) (*Response, error)
	Disable(ctx context.Context, id string) (*Response, error)
}

type ListRequest struct {
	Name      string
	Code      string
	IsEnabled *bool
	SortBy    string
	OrderBy   string
}

type CreateRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	TaxMode     TaxMode  `json:"tax_mode"`
	Rate        *float64 `json:"rate"`
	Description *string  `json:"description"`
	IsEnabled   *bool    `json:"is_enabled"`
}

type UpdateRequest struct {
	ID          string   `json:"id"`
	Name        *string  `json:"name,omitempty"`
	TaxMode     *TaxMode `json:"tax_mode,omitempty"`
	Rate        *float64 `json:"rate,omitempty"`
	Description *string  `json:"description,omitempty"`
}

type Response struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	TaxMode        TaxMode   `json:"tax_mode"`
	Rate           *float64  `json:"rate,omitempty"`
	Description    *string   `json:"description,omitempty"`
	IsEnabled      bool      `json:"is_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
