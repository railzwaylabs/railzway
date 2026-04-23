// Package domain contains persistence models for rating results and aggregates.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RatingResult struct {
	ID              uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID           uuid.UUID       `gorm:"not null;index" json:"org_id"`
	UsageEventID    uuid.UUID       `gorm:"not null;index" json:"usage_event_id"`
	CustomerID      uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	SubscriptionID  *uuid.UUID      `gorm:"index" json:"subscription_id,omitempty"`
	PlanPriceID     uuid.UUID       `gorm:"not null;index" json:"plan_price_id"`
	PlanAmountID    *uuid.UUID      `gorm:"index" json:"plan_amount_id,omitempty"`
	MeterID         uuid.UUID       `gorm:"not null;index" json:"meter_id"`
	Currency        string          `gorm:"type:text;not null" json:"currency"`
	Quantity        float64         `gorm:"not null" json:"quantity"`
	UnitAmountCents float64         `gorm:"type:numeric(28,12);not null" json:"unit_amount_cents"`
	AmountCents     int64           `gorm:"not null" json:"amount_cents"`
	Source          string          `gorm:"type:text;not null" json:"source"`
	WindowStart     time.Time       `gorm:"not null" json:"window_start"`
	WindowEnd       time.Time       `gorm:"not null" json:"window_end"`
	Metadata        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (RatingResult) TableName() string { return "rating_results" }

type UsageAggregate struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CustomerID     uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	SubscriptionID *uuid.UUID      `gorm:"index" json:"subscription_id,omitempty"`
	PlanPriceID    uuid.UUID       `gorm:"not null;index" json:"plan_price_id"`
	PlanAmountID   *uuid.UUID      `gorm:"index" json:"plan_amount_id,omitempty"`
	MeterID        uuid.UUID       `gorm:"not null;index" json:"meter_id"`
	Currency       string          `gorm:"type:text;not null" json:"currency"`
	PeriodStart    time.Time       `gorm:"not null" json:"period_start"`
	PeriodEnd      time.Time       `gorm:"not null" json:"period_end"`
	Quantity       float64         `gorm:"not null" json:"quantity"`
	AmountCents    int64           `gorm:"not null" json:"amount_cents"`
	LastEventAt    *time.Time      `gorm:"" json:"last_event_at,omitempty"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (UsageAggregate) TableName() string { return "usage_aggregates" }
