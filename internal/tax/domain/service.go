package domain

import (
	"context"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

type ListTaxRatesRequest struct {
	PageToken   string
	PageSize    int32
	Code        string
	Name        string
	Active      *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type ListTaxRatesResponse struct {
	pagination.PageInfo
	Rates []TaxRate `json:"rates"`
}

type CreateTaxRateRequest struct {
	Code       string
	Name       string
	Percentage float64
	Inclusive  bool
	Active     bool
	Metadata   []byte
}

type Service interface {
	ListTaxRates(ctx context.Context, req ListTaxRatesRequest) (ListTaxRatesResponse, error)
	CreateTaxRate(ctx context.Context, req CreateTaxRateRequest) (TaxRate, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidCursor       = errors.New("invalid_cursor")
	ErrInvalidCode         = errors.New("invalid_code")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidPercentage   = errors.New("invalid_percentage")
	ErrTaxCodeExists       = errors.New("tax_code_exists")
)
