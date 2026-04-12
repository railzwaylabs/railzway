package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

const (
	AggregationSum   = "sum"
	AggregationCount = "count"
	AggregationMax   = "max"
	AggregationLast  = "last"
	AggregationAvg   = "avg"

	StatusAccepted = "accepted"
	StatusEnriched = "enriched"
	StatusRated    = "rated"
)

type CreateMeterRequest struct {
	Code           string
	Name           string
	Aggregation    string
	Unit           string
	Active         *bool
	IdempotencyKey string
}

type UpdateMeterRequest struct {
	Name        *string
	Aggregation *string
	Unit        *string
	Active      *bool
}

type GetMeterRequest struct {
	ID string
}

type ListMeterRequest struct {
	PageToken string
	PageSize  int32
	Code      string
	Name      string
	Active    *bool
}

type MeterResponse struct {
	ID          string          `json:"id"`
	OrgID       string          `json:"org_id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Aggregation string          `json:"aggregation"`
	Unit        string          `json:"unit"`
	Active      bool            `json:"active"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ListMeterResponse struct {
	pagination.PageInfo
	Meters []MeterResponse `json:"meters"`
}

type IngestUsageRequest struct {
	MeterCode      string
	CustomerID     string
	Value          float64
	RecordedAt     time.Time
	IdempotencyKey string
}

type UsageEventResponse struct {
	ID         string          `json:"id"`
	OrgID      string          `json:"org_id"`
	MeterID    string          `json:"meter_id"`
	MeterCode  string          `json:"meter_code"`
	CustomerID string          `json:"customer_id"`
	Value      float64         `json:"value"`
	RecordedAt time.Time       `json:"recorded_at"`
	Status     string          `json:"status"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ListUsageRequest struct {
	PageToken    string
	PageSize     int32
	MeterID      string
	CustomerID   string
	Status       string
	RecordedFrom *time.Time
	RecordedTo   *time.Time
}

type ListUsageResponse struct {
	pagination.PageInfo
	Events []UsageEventResponse `json:"events"`
}

type Service interface {
	CreateMeter(ctx context.Context, req CreateMeterRequest) (MeterResponse, error)
	UpdateMeter(ctx context.Context, id string, req UpdateMeterRequest) (MeterResponse, error)
	GetMeterByID(ctx context.Context, req GetMeterRequest) (MeterResponse, error)
	ListMeters(ctx context.Context, req ListMeterRequest) (ListMeterResponse, error)

	IngestUsage(ctx context.Context, req IngestUsageRequest) (UsageEventResponse, error)
	ListUsage(ctx context.Context, req ListUsageRequest) (ListUsageResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidCode         = errors.New("invalid_code")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidAggregation  = errors.New("invalid_aggregation")
	ErrInvalidUnit         = errors.New("invalid_unit")
	ErrInvalidMeter        = errors.New("invalid_meter")
	ErrInvalidCustomer     = errors.New("invalid_customer")
	ErrInvalidValue        = errors.New("invalid_value")
	ErrInvalidRecordedAt   = errors.New("invalid_recorded_at")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrNotFound            = errors.New("not_found")
)
