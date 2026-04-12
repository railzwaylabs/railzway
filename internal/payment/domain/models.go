// Package domain contains payment persistence models.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PaymentMethod struct {
	ID         uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID      uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CustomerID uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	Provider   string          `gorm:"type:text;not null" json:"provider"`
	MethodType string          `gorm:"type:text;not null" json:"method_type"`
	Last4      *string         `gorm:"type:text" json:"last4,omitempty"`
	ExpMonth   *int            `gorm:"" json:"exp_month,omitempty"`
	ExpYear    *int            `gorm:"" json:"exp_year,omitempty"`
	Metadata   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt  time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PaymentMethod) TableName() string { return "payment_methods" }

type Payment struct {
	ID              uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID           uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CustomerID      uuid.UUID       `gorm:"not null;index" json:"customer_id"`
	InvoiceID       *uuid.UUID      `gorm:"index" json:"invoice_id,omitempty"`
	PaymentMethodID *uuid.UUID      `gorm:"index" json:"payment_method_id,omitempty"`
	Provider        string          `gorm:"type:text;not null" json:"provider"`
	ProviderRef     *string         `gorm:"type:text" json:"provider_ref,omitempty"`
	Status          string          `gorm:"type:text;not null" json:"status"`
	AmountCents     int64           `gorm:"not null" json:"amount_cents"`
	Currency        string          `gorm:"type:text;not null" json:"currency"`
	IdempotencyKey  *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Payment) TableName() string { return "payments" }

type Refund struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	PaymentID      uuid.UUID       `gorm:"not null;index" json:"payment_id"`
	ProviderRef    *string         `gorm:"type:text" json:"provider_ref,omitempty"`
	Status         string          `gorm:"type:text;not null" json:"status"`
	AmountCents    int64           `gorm:"not null" json:"amount_cents"`
	Currency       string          `gorm:"type:text;not null" json:"currency"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (Refund) TableName() string { return "refunds" }
