// Package domain contains persistence models for invoices.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Invoice struct {
	ID              uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID           uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CustomerID      uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	SubscriptionID  *uuid.UUID      `gorm:"index" json:"subscription_id,omitempty"`
	Number          string          `gorm:"type:text;not null" json:"number"`
	Status          string          `gorm:"type:text;not null" json:"status"`
	Currency        string          `gorm:"type:text;not null" json:"currency"`
	SubtotalCents   int64           `gorm:"not null" json:"subtotal_cents"`
	TaxCents        int64           `gorm:"not null" json:"tax_cents"`
	TotalCents      int64           `gorm:"not null" json:"total_cents"`
	AmountDueCents  int64           `gorm:"not null" json:"amount_due_cents"`
	AmountPaidCents int64           `gorm:"not null" json:"amount_paid_cents"`
	Checksum        string          `gorm:"type:text;not null;default:''" json:"checksum"`
	PeriodStart     time.Time       `gorm:"not null" json:"period_start"`
	PeriodEnd       time.Time       `gorm:"not null" json:"period_end"`
	IssuedAt        *time.Time      `gorm:"" json:"issued_at,omitempty"`
	DueAt           *time.Time      `gorm:"" json:"due_at,omitempty"`
	PaidAt          *time.Time      `gorm:"" json:"paid_at,omitempty"`
	VoidedAt        *time.Time      `gorm:"" json:"voided_at,omitempty"`
	IdempotencyKey  *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Invoice) TableName() string { return "invoices" }

type InvoiceItem struct {
	ID              uuid.UUID       `gorm:"primaryKey" json:"id"`
	InvoiceID       uuid.UUID       `gorm:"not null;index" json:"invoice_id"`
	OrgID           uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CustomerID      uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	SubscriptionID  *uuid.UUID      `gorm:"index" json:"subscription_id,omitempty"`
	PlanPriceID     *uuid.UUID      `gorm:"index" json:"plan_price_id,omitempty"`
	MeterID         *uuid.UUID      `gorm:"index" json:"meter_id,omitempty"`
	RatingResultID  *uuid.UUID      `gorm:"index" json:"rating_result_id,omitempty"`
	LineType        string          `gorm:"type:text;not null" json:"line_type"`
	Description     string          `gorm:"type:text" json:"description,omitempty"`
	Quantity        float64         `gorm:"not null" json:"quantity"`
	UnitAmountCents int64           `gorm:"not null" json:"unit_amount_cents"`
	AmountCents     int64           `gorm:"not null" json:"amount_cents"`
	Currency        string          `gorm:"type:text;not null" json:"currency"`
	PeriodStart     *time.Time      `gorm:"" json:"period_start,omitempty"`
	PeriodEnd       *time.Time      `gorm:"" json:"period_end,omitempty"`
	IdempotencyKey  *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (InvoiceItem) TableName() string { return "invoice_items" }
