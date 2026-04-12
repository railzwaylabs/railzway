package domain

import (
	"context"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

const (
	StatusDraft         = "draft"
	StatusOpen          = "open"
	StatusPaid          = "paid"
	StatusVoid          = "void"
	StatusUncollectible = "uncollectible"

	LineTypeSubscription = "subscription"
	LineTypeUsage        = "usage"
	LineTypeAdjustment   = "adjustment"
	LineTypeCredit       = "credit"
	LineTypeTax          = "tax"
)

type GenerateInvoiceRequest struct {
	SubscriptionID string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	IssueAt        *time.Time
	DueAt          *time.Time
	IdempotencyKey string
}

type GenerateInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type GetInvoiceRequest struct {
	ID string
}

type GetInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type OpenInvoiceRequest struct {
	ID string
}

type OpenInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type RecalculateDraftInvoiceRequest struct {
	ID string
}

type RecalculateDraftInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type PayInvoiceRequest struct {
	ID string
}

type PayInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type VoidInvoiceRequest struct {
	ID string
}

type VoidInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type ResendInvoiceRequest struct {
	ID string
}

type ResendInvoiceResponse struct {
	Status string `json:"status"`
}

type ListInvoicesRequest struct {
	PageToken       string
	PageSize        int32
	CustomerID      string
	SubscriptionID  string
	Status          string
	Number          string
	PeriodStartFrom *time.Time
	PeriodStartTo   *time.Time
	IssuedFrom      *time.Time
	IssuedTo        *time.Time
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
}

type ListInvoicesResponse struct {
	pagination.PageInfo
	Invoices []Invoice `json:"invoices"`
}

type AdjustmentLine struct {
	RatingResultID  *string
	PlanPriceID     *string
	MeterID         *string
	Quantity        float64
	UnitAmountCents int64
	AmountCents     int64
	Currency        string
	Description     string
	WindowStart     *time.Time
	WindowEnd       *time.Time
}

type CreateAdjustmentInvoiceRequest struct {
	SubscriptionID string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	BaseInvoiceID  string
	Reason         string
	Lines          []AdjustmentLine
	IssueAt        *time.Time
	DueAt          *time.Time
	IdempotencyKey string
}

type CreateAdjustmentInvoiceResponse struct {
	Invoice Invoice       `json:"invoice"`
	Items   []InvoiceItem `json:"items"`
}

type Service interface {
	GenerateInvoice(ctx context.Context, req GenerateInvoiceRequest) (GenerateInvoiceResponse, error)
	CreateAdjustmentInvoice(ctx context.Context, req CreateAdjustmentInvoiceRequest) (CreateAdjustmentInvoiceResponse, error)
	RecalculateDraftInvoice(ctx context.Context, req RecalculateDraftInvoiceRequest) (RecalculateDraftInvoiceResponse, error)
	GetInvoice(ctx context.Context, req GetInvoiceRequest) (GetInvoiceResponse, error)
	OpenInvoice(ctx context.Context, req OpenInvoiceRequest) (OpenInvoiceResponse, error)
	PayInvoice(ctx context.Context, req PayInvoiceRequest) (PayInvoiceResponse, error)
	VoidInvoice(ctx context.Context, req VoidInvoiceRequest) (VoidInvoiceResponse, error)
	ResendInvoice(ctx context.Context, req ResendInvoiceRequest) (ResendInvoiceResponse, error)
	ListInvoices(ctx context.Context, req ListInvoicesRequest) (ListInvoicesResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidSubscription = errors.New("invalid_subscription")
	ErrInvalidCustomer     = errors.New("invalid_customer")
	ErrInvalidPeriod       = errors.New("invalid_period")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrInvalidCursor       = errors.New("invalid_cursor")
	ErrNotFound            = errors.New("not_found")
	ErrUsageNotReady       = errors.New("usage_not_ready")
)
