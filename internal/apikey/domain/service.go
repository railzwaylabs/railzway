package domain

import (
	"context"
	"errors"
	"time"
)

type CreateAPIKeyRequest struct {
	Name           string   `json:"name"`
	KeyType        string   `json:"key_type"`
	Scopes         []string `json:"scopes,omitempty"`
	AllowedIPs     []string `json:"allowed_ips,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
}

type APIKeyResponse struct {
	ID             string     `json:"id"`
	OrgID          string     `json:"org_id"`
	Name           string     `json:"name"`
	KeyPrefix      string     `json:"key_prefix"`
	KeyType        string     `json:"key_type"`
	Scopes         []string   `json:"scopes"`
	AllowedIPs     []string   `json:"allowed_ips"`
	AllowedDomains []string   `json:"allowed_domains"`
	Status         string     `json:"status"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Key            string     `json:"key,omitempty"`
}

type ListAPIKeysResponse struct {
	Keys []APIKeyResponse `json:"keys"`
}

type AuthorizeAPIKeyRequest struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	IP       string `json:"ip"`
	Domain   string `json:"domain"`
}

type Service interface {
	ListKeys(ctx context.Context, orgID string) (ListAPIKeysResponse, error)
	CreateKey(ctx context.Context, orgID string, req CreateAPIKeyRequest) (APIKeyResponse, error)
	RevokeKey(ctx context.Context, orgID string, id string) (APIKeyResponse, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidKeyType      = errors.New("invalid_key_type")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidKey          = errors.New("invalid_key")
	ErrNotFound            = errors.New("not_found")
	ErrKeyRevoked          = errors.New("key_revoked")
	ErrKeyNotAllowed       = errors.New("key_not_allowed")
	ErrScopeForbidden      = errors.New("scope_forbidden")
)
