package domain

import (
	"context"
	"errors"
)

type Service interface {
	List(ctx context.Context, req ListRequest) ([]Response, error)
	Replace(ctx context.Context, req ReplaceRequest) ([]Response, error)
	ListForProducts(ctx context.Context, req ListForProductsRequest) ([]Snapshot, error)
}

type ListRequest struct {
	ProductID string
}

type ReplaceRequest struct {
	ProductID  string
	FeatureIDs []string
}

type ListForProductsRequest struct {
	ProductIDs []string
}

type Response struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	FeatureType string  `json:"feature_type"`
	MeterID     *string `json:"meter_id,omitempty"`
	Active      bool    `json:"active"`
}

type Snapshot struct {
	FeatureID   string
	Code        string
	Name        string
	FeatureType string
	MeterID     *string
	Active      bool
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidProductID    = errors.New("invalid_product_id")
	ErrInvalidFeatureID    = errors.New("invalid_feature_id")
	ErrInvalidMeterID      = errors.New("invalid_meter_id")
	ErrProductNotFound     = errors.New("product_not_found")
	ErrFeatureNotFound     = errors.New("feature_not_found")
	ErrFeatureInactive     = errors.New("feature_inactive")
	ErrMeterNotFound       = errors.New("meter_not_found")
)
