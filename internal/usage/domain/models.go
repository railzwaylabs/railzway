// Package domain contains persistence models for meters and usage events.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Meter struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	Code           string          `gorm:"type:text;not null" json:"code"`
	Name           string          `gorm:"type:text;not null" json:"name"`
	Aggregation    string          `gorm:"type:text;not null" json:"aggregation"`
	Unit           string          `gorm:"type:text;not null" json:"unit"`
	Active         bool            `gorm:"not null;default:true" json:"active"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Meter) TableName() string { return "meters" }

type UsageEvent struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	MeterID        uuid.UUID       `gorm:"not null;index" json:"meter_id"`
	MeterCode      string          `gorm:"type:text;not null" json:"meter_code"`
	CustomerID     uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	Value          float64         `gorm:"not null" json:"value"`
	RecordedAt     time.Time       `gorm:"not null" json:"recorded_at"`
	Status         string          `gorm:"type:text;not null" json:"status"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (UsageEvent) TableName() string { return "usage_events" }
