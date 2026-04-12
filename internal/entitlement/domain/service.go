package domain

import (
	"context"
	"errors"
)

type CheckEntitlementRequest struct {
	CustomerID  string
	FeatureCode string
}

type EntitlementResponse struct {
	HasAccess bool    `json:"has_access"`
	IsMetered bool    `json:"is_metered"`
	Usage     float64 `json:"usage,omitempty"`
	Limit     float64 `json:"limit,omitempty"`
}

type Service interface {
	CheckEntitlement(ctx context.Context, req CheckEntitlementRequest) (EntitlementResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidCustomer     = errors.New("invalid_customer")
	ErrInvalidFeature      = errors.New("invalid_feature")
)
