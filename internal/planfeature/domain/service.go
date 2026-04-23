package domain

import (
	"context"
	"errors"
)

type Service interface {
	List(ctx context.Context, req ListRequest) ([]Response, error)
	Replace(ctx context.Context, req ReplaceRequest) ([]Response, error)
	ListForPlans(ctx context.Context, req ListForPlansRequest) ([]Snapshot, error)
}

type ListRequest struct {
	PlanID string
}

type ReplaceRequest struct {
	PlanID   string
	Features []ReplaceFeatureInput
}

type ReplaceFeatureInput struct {
	FeatureID    string   `json:"feature_id"`
	Enabled      bool     `json:"enabled"`
	LimitNumeric *float64 `json:"limit_numeric,omitempty"`
	LimitUnit    *string  `json:"limit_unit,omitempty"`
	ResetPeriod  *string  `json:"reset_period,omitempty"`
}

type ListForPlansRequest struct {
	PlanIDs []string
}

type Response struct {
	ID           string   `json:"id"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	FeatureType  string   `json:"feature_type"`
	MeterID      *string  `json:"meter_id,omitempty"`
	Active       bool     `json:"active"`
	Enabled      bool     `json:"enabled"`
	LimitNumeric *float64 `json:"limit_numeric,omitempty"`
	LimitUnit    *string  `json:"limit_unit,omitempty"`
	ResetPeriod  string   `json:"reset_period"`
}

type Snapshot struct {
	PlanID       string
	FeatureID    string
	Code         string
	Name         string
	FeatureType  string
	MeterID      *string
	Active       bool
	Enabled      bool
	LimitNumeric *float64
	LimitUnit    *string
	ResetPeriod  string
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidPlanID       = errors.New("invalid_plan_id")
	ErrInvalidFeatureID    = errors.New("invalid_feature_id")
	ErrInvalidMeterID      = errors.New("invalid_meter_id")
	ErrInvalidLimit        = errors.New("invalid_limit")
	ErrInvalidResetPeriod  = errors.New("invalid_reset_period")
	ErrPlanNotFound        = errors.New("plan_not_found")
	ErrFeatureNotFound     = errors.New("feature_not_found")
	ErrFeatureInactive     = errors.New("feature_inactive")
	ErrFeatureNotAllowed   = errors.New("feature_not_allowed")
	ErrMeterNotFound       = errors.New("meter_not_found")
)
