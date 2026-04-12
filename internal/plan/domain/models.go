// Package domain contains persistence models for plans and pricing.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	Code           string          `gorm:"type:text;not null" json:"code"`
	Name           string          `gorm:"type:text;not null" json:"name"`
	Description    string          `gorm:"type:text" json:"description,omitempty"`
	Active         bool            `gorm:"not null;default:true" json:"active"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	ProductID      *uuid.UUID      `gorm:"column:product_id" json:"product_id,omitempty"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Plan) TableName() string { return "plans" }

type PlanPrice struct {
	ID                   uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID                uuid.UUID       `gorm:"not null;index" json:"org_id"`
	PlanID               uuid.UUID       `gorm:"not null;index" json:"plan_id"`
	MeterID              *uuid.UUID      `gorm:"column:meter_id" json:"meter_id,omitempty"`
	Code                 string          `gorm:"type:text;not null" json:"code"`
	Name                 string          `gorm:"type:text" json:"name,omitempty"`
	Description          string          `gorm:"type:text" json:"description,omitempty"`
	PriceType            string          `gorm:"type:text;not null" json:"price_type"`
	BillingInterval      string          `gorm:"type:text;not null" json:"billing_interval"`
	BillingIntervalCount int             `gorm:"not null;default:1" json:"billing_interval_count"`
	AggregateUsage       string          `gorm:"type:text" json:"aggregate_usage,omitempty"`
	BillingUnit          string          `gorm:"type:text" json:"billing_unit,omitempty"`
	MeterCode            string          `gorm:"type:text" json:"meter_code,omitempty"`
	Active               bool            `gorm:"not null;default:true" json:"active"`
	IdempotencyKey       *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata             json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt            time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlanPrice) TableName() string { return "plan_prices" }

type PlanAmount struct {
	ID                 uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID              uuid.UUID       `gorm:"not null;index" json:"org_id"`
	PlanPriceID        uuid.UUID       `gorm:"not null;index" json:"plan_price_id"`
	Currency           string          `gorm:"type:text;not null" json:"currency"`
	UnitAmountCents    int64           `gorm:"not null" json:"unit_amount_cents"`
	MinimumAmountCents *int64          `gorm:"column:minimum_amount_cents" json:"minimum_amount_cents,omitempty"`
	MaximumAmountCents *int64          `gorm:"column:maximum_amount_cents" json:"maximum_amount_cents,omitempty"`
	EffectiveFrom      time.Time       `gorm:"not null" json:"effective_from"`
	EffectiveTo        *time.Time      `gorm:"" json:"effective_to,omitempty"`
	IdempotencyKey     *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata           json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt          time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlanAmount) TableName() string { return "plan_amounts" }

type PlanTier struct {
	ID              uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID           uuid.UUID       `gorm:"not null;index" json:"org_id"`
	PlanPriceID     uuid.UUID       `gorm:"not null;index" json:"plan_price_id"`
	TierMode        string          `gorm:"type:text;not null" json:"tier_mode"`
	StartQuantity   float64         `gorm:"type:numeric;not null" json:"start_quantity"`
	EndQuantity     *float64        `gorm:"type:numeric" json:"end_quantity,omitempty"`
	UnitAmountCents *int64          `gorm:"column:unit_amount_cents" json:"unit_amount_cents,omitempty"`
	FlatAmountCents *int64          `gorm:"column:flat_amount_cents" json:"flat_amount_cents,omitempty"`
	Unit            string          `gorm:"type:text;not null" json:"unit"`
	IdempotencyKey  *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlanTier) TableName() string { return "plan_tiers" }
