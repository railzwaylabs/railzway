package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

type Customer struct {
	ID        snowflake.ID      `gorm:"primaryKey" json:"id"`
	OrgID     snowflake.ID      `gorm:"not null;index" json:"organization_id"`
	Name      string            `gorm:"not null" json:"name"`
	Email     string            `gorm:"not null" json:"email"`
	Currency  string            `gorm:"column:currency" json:"currency,omitempty"`
	Metadata  datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}
