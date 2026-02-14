package domain

import (
	"context"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Response, error)
	List(ctx context.Context, req ListRequest) (ListResponse, error)
	Update(ctx context.Context, req UpdateRequest) (*Response, error)
	Archive(ctx context.Context, id string) (*Response, error)
}

type ListRequest struct {
	Name        string
	Code        string
	FeatureType *FeatureType
	Active      *bool
	SortBy      string
	OrderBy     string
	PageToken   string
	PageSize    int32
}

type ListResponse struct {
	PageInfo pagination.PageInfo `json:"page_info"`
	Features []Response          `json:"features"`
}

type CreateRequest struct {
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	FeatureType FeatureType    `json:"feature_type"`
	MeterID     *string        `json:"meter_id"`
	Active      *bool          `json:"active"`
	Metadata    map[string]any `json:"metadata"`
	IdempotencyKey string       `json:"-"`
}

type UpdateRequest struct {
	ID          string         `json:"id"`
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	FeatureType *FeatureType   `json:"feature_type,omitempty"`
	MeterID     *string        `json:"meter_id,omitempty"`
	Active      *bool          `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Response struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	Description    *string        `json:"description,omitempty"`
	FeatureType    FeatureType    `json:"feature_type"`
	MeterID        *string        `json:"meter_id,omitempty"`
	Active         bool           `json:"active"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidCode         = errors.New("invalid_code")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidType         = errors.New("invalid_feature_type")
	ErrInvalidMeterID      = errors.New("invalid_meter_id")
	ErrInvalidID           = errors.New("invalid_id")
	ErrNotFound            = errors.New("not_found")
)
