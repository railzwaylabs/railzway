package oauth2provider

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// AuthorizationCode stores an issued OAuth2 authorization code.
type AuthorizationCode struct {
	CodeHash            string       `gorm:"column:code_hash;type:text;primaryKey"`
	ClientID            string       `gorm:"column:client_id;type:text;not null;index"`
	RedirectURI         string       `gorm:"column:redirect_uri;type:text;not null"`
	UserID              snowflake.ID `gorm:"column:user_id;not null;index"`
	Scopes              []string     `gorm:"column:scopes;type:jsonb;serializer:json"`
	CodeChallengeHash   *string      `gorm:"column:code_challenge_hash;type:text"`
	CodeChallengeMethod *string      `gorm:"column:code_challenge_method;type:text"`
	ExpiresAt           time.Time    `gorm:"column:expires_at;not null;index"`
	UsedAt              *time.Time   `gorm:"column:used_at"`
	CreatedAt           time.Time    `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
}

func (AuthorizationCode) TableName() string { return "oauth_authorization_codes" }

// AccessToken stores an issued OAuth2 access token.
type AccessToken struct {
	TokenHash string       `gorm:"column:token_hash;type:text;primaryKey"`
	ClientID  string       `gorm:"column:client_id;type:text;not null;index"`
	UserID    snowflake.ID `gorm:"column:user_id;not null;index"`
	Scopes    []string     `gorm:"column:scopes;type:jsonb;serializer:json"`
	ExpiresAt time.Time    `gorm:"column:expires_at;not null;index"`
	RevokedAt *time.Time   `gorm:"column:revoked_at"`
	CreatedAt time.Time    `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
}

func (AccessToken) TableName() string { return "oauth_access_tokens" }
