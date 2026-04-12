package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

type CreateFeatureRequest struct {
	Code           string
	Name           string
	Description    *string
	FeatureType    string
	MeterID        *string
	Active         *bool
	IdempotencyKey string
}

type UpdateFeatureRequest struct {
	Name        *string
	Description *string
	FeatureType *string
	MeterID     *string
	Active      *bool
}

type GetFeatureRequest struct {
	ID string
}

type ListFeatureRequest struct {
	PageToken   string
	PageSize    int32
	Code        string
	Name        string
	FeatureType string
	Active      *bool
}

type FeatureResponse struct {
	ID          string          `json:"id"`
	OrgID       string          `json:"org_id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	FeatureType string          `json:"feature_type"`
	MeterID     *string         `json:"meter_id,omitempty"`
	Active      bool            `json:"active"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ListFeatureResponse struct {
	pagination.PageInfo
	Features []FeatureResponse `json:"features"`
}

type Service interface {
	Create(ctx context.Context, req CreateFeatureRequest) (FeatureResponse, error)
	Update(ctx context.Context, id string, req UpdateFeatureRequest) (FeatureResponse, error)
	GetByID(ctx context.Context, req GetFeatureRequest) (FeatureResponse, error)
	List(ctx context.Context, req ListFeatureRequest) (ListFeatureResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidCode         = errors.New("invalid_code")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidType         = errors.New("invalid_type")
	ErrInvalidMeter        = errors.New("invalid_meter")
	ErrNotFound            = errors.New("not_found")
	ErrInactive            = errors.New("inactive")
)
