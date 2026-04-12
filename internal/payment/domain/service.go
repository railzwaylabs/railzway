package domain

import (
	"context"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"
)

type ListPaymentsRequest struct {
	PageToken   string
	PageSize    int32
	CustomerID  string
	InvoiceID   string
	Status      string
	Provider    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type ListPaymentsResponse struct {
	pagination.PageInfo
	Payments []Payment `json:"payments"`
}

type Service interface {
	ListPayments(ctx context.Context, req ListPaymentsRequest) (ListPaymentsResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidCustomer     = errors.New("invalid_customer")
	ErrInvalidInvoice      = errors.New("invalid_invoice")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrInvalidCursor       = errors.New("invalid_cursor")
)
