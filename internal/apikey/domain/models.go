package domain

import "time"

type KeyType string

const (
	KeyTypePublic  KeyType = "public"
	KeyTypeSecret  KeyType = "secret"
	KeyTypeWebhook KeyType = "webhook"
)

type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRevoked  KeyStatus = "revoked"
	KeyStatusDisabled KeyStatus = "disabled"
)

type APIKey struct {
	ID            string    `json:"id"`
	OrgID         string    `json:"org_id"`
	Name          string    `json:"name"`
	KeyPrefix     string    `json:"key_prefix"`
	KeyType       KeyType   `json:"key_type"`
	Scopes        []string  `json:"scopes"`
	AllowedIPs    []string  `json:"allowed_ips"`
	AllowedDomain []string  `json:"allowed_domains"`
	Status        KeyStatus `json:"status"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
