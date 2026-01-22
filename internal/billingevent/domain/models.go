package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

// BillingEvent captures outbox events for billing workflows.
type BillingEvent struct {
	ID          snowflake.ID      `gorm:"primaryKey"`
	OrgID       snowflake.ID      `gorm:"not null;index;uniqueIndex:ux_billing_event_dedupe,priority:1"`
	EventType   string            `gorm:"type:text;not null"`
	Payload     datatypes.JSONMap `gorm:"type:jsonb;not null"`
	DedupeKey   *string           `gorm:"type:text;uniqueIndex:ux_billing_event_dedupe,priority:2"`
	Published   bool              `gorm:"not null;default:false"`
	PublishedAt *time.Time        `gorm:""`
	CreatedAt   time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName sets the database table name.
func (BillingEvent) TableName() string { return "billing_events" }
