package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

type CreateCustomerRequest struct {
	Name           string
	Email          string
	ExternalID     string
	Currency       string
	TestClockID    *string
	IdempotencyKey string
}

type UpdateCustomerRequest struct {
	Name        *string
	Email       *string
	ExternalID  *string
	Currency    *string
	TestClockID *string
}

type GetCustomerRequest struct {
	ID string
}

type ListCustomerRequest struct {
	PageToken   string
	PageSize    int32
	Name        string
	Email       string
	Currency    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type ListCustomerResponse struct {
	pagination.PageInfo
	Customers []CustomerResponse `json:"customers"`
}

type CustomerResponse struct {
	ID          string          `json:"id"`
	OrgID       string          `json:"org_id"`
	TestClockID *string         `json:"test_clock_id,omitempty"`
	ExternalID  string          `json:"external_id,omitempty"`
	Name        string          `json:"name"`
	Email       string          `json:"email"`
	Currency    string          `json:"currency,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Service interface {
	Create(ctx context.Context, req CreateCustomerRequest) (CustomerResponse, error)
	Update(ctx context.Context, id string, req UpdateCustomerRequest) (CustomerResponse, error)
	GetByID(ctx context.Context, req GetCustomerRequest) (CustomerResponse, error)
	List(ctx context.Context, req ListCustomerRequest) (ListCustomerResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidEmail        = errors.New("invalid_email")
	ErrInvalidCurrency     = errors.New("invalid_currency")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidTestClock    = errors.New("invalid_test_clock")
	ErrNotFound            = errors.New("not_found")
)
