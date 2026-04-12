// Package domain contains persistence models for subscriptions.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID                 uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID              uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CustomerID         uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	PlanID             uuid.UUID       `gorm:"not null;index" json:"plan_id"`
	Status             string          `gorm:"type:text;not null" json:"status"`
	Currency           string          `gorm:"type:text;not null" json:"currency"`
	StartAt            time.Time       `gorm:"not null" json:"start_at"`
	CurrentPeriodStart time.Time       `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd   time.Time       `gorm:"not null" json:"current_period_end"`
	TrialEnd           *time.Time      `gorm:"" json:"trial_end,omitempty"`
	CancelAt           *time.Time      `gorm:"" json:"cancel_at,omitempty"`
	CanceledAt         *time.Time      `gorm:"" json:"canceled_at,omitempty"`
	EndedAt            *time.Time      `gorm:"" json:"ended_at,omitempty"`
	IdempotencyKey     *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata           json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt          time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Subscription) TableName() string { return "subscriptions" }

type SubscriptionItem struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	SubscriptionID uuid.UUID       `gorm:"not null;index" json:"subscription_id"`
	PlanPriceID    uuid.UUID       `gorm:"not null;index" json:"plan_price_id"`
	Quantity       float64         `gorm:"not null;default:1" json:"quantity"`
	StartAt        time.Time       `gorm:"not null" json:"start_at"`
	EndAt          *time.Time      `gorm:"" json:"end_at,omitempty"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (SubscriptionItem) TableName() string { return "subscription_items" }

type SubscriptionPeriod struct {
	ID             uuid.UUID `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID `gorm:"not null;index" json:"org_id"`
	SubscriptionID uuid.UUID `gorm:"not null;index" json:"subscription_id"`
	Status         string    `gorm:"type:text;not null" json:"status"`
	PeriodStart    time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd      time.Time `gorm:"not null" json:"period_end"`
	CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (SubscriptionPeriod) TableName() string { return "subscription_periods" }
