// Package domain contains persistence models for the customer service.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"organization_id"`
	TestClockID    *uuid.UUID      `gorm:"index" json:"test_clock_id,omitempty"`
	ExternalID     string          `gorm:"column:external_id" json:"external_id,omitempty"`
	Name           string          `gorm:"not null" json:"name"`
	Email          string          `gorm:"not null" json:"email"`
	Currency       string          `gorm:"column:currency" json:"currency,omitempty"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the database table name.
func (Customer) TableName() string { return "customers" }
