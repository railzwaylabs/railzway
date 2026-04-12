package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type AppCategory string

const (
	CategoryPayment AppCategory = "payment"
	CategoryEmail   AppCategory = "email"
)

type AppStatus string

const (
	StatusActive     AppStatus = "active"
	StatusComingSoon AppStatus = "coming_soon"
)

type InstallationStatus string

const (
	InstallStatusActive   InstallationStatus = "active"
	InstallStatusDisabled InstallationStatus = "disabled"
)

type AppDefinition struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Category          AppCategory         `json:"category"`
	Provider          string              `json:"provider"`
	Description       string              `json:"description"`
	Capabilities      []string            `json:"capabilities"`
	AuthMethods       []string            `json:"auth_methods"`
	CredentialsSchema map[string][]string `json:"credentials_schema,omitempty"`
	Status            AppStatus           `json:"status"`
	Version           string              `json:"version"`
}

type Warning struct {
	Module  string `json:"module,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ListAppsResponse struct {
	Apps     []AppDefinition `json:"apps"`
	Warnings []Warning       `json:"warnings,omitempty"`
}

type AppInstallation struct {
	ID          uuid.UUID          `json:"id"`
	OrgID       uuid.UUID          `json:"org_id"`
	AppID       string             `json:"app_id"`
	Status      InstallationStatus `json:"status"`
	AuthMethod  string             `json:"auth_method"`
	Config      json.RawMessage    `json:"config"`
	Credentials json.RawMessage    `json:"credentials"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type ListInstallationsResponse struct {
	Installations []AppInstallation `json:"installations"`
	Warnings      []Warning         `json:"warnings,omitempty"`
}

type InstallAppRequest struct {
	AppID       string          `json:"app_id"`
	AuthMethod  string          `json:"auth_method,omitempty"`
	Config      json.RawMessage `json:"config,omitempty"`
	Credentials json.RawMessage `json:"credentials,omitempty"`
}

type UpdateInstallationRequest struct {
	Status      *string         `json:"status,omitempty"`
	AuthMethod  *string         `json:"auth_method,omitempty"`
	Config      json.RawMessage `json:"config,omitempty"`
	Credentials json.RawMessage `json:"credentials,omitempty"`
}

type Service interface {
	ListCatalog(ctx context.Context) (ListAppsResponse, error)
	ListInstallations(ctx context.Context) (ListInstallationsResponse, error)
	InstallApp(ctx context.Context, req InstallAppRequest) (AppInstallation, error)
	UpdateInstallation(ctx context.Context, id string, req UpdateInstallationRequest) (AppInstallation, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidApp          = errors.New("invalid_app")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrInvalidAuthMethod   = errors.New("invalid_auth_method")
	ErrInvalidID           = errors.New("invalid_id")
	ErrNotFound            = errors.New("not_found")
	ErrCredentialsKey      = errors.New("credentials_key_missing_or_invalid")
	ErrMissingCredentials  = errors.New("missing_credentials")
)
