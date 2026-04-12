// Package domain contains persistence models for the ledger.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LedgerEntryType string

const (
	LedgerEntryTypeDebit  LedgerEntryType = "debit"
	LedgerEntryTypeCredit LedgerEntryType = "credit"
)

type LedgerSourceType string

const (
	SourceTypeBillingCycle LedgerSourceType = "billing_cycle"
	SourceTypeAdjustment   LedgerSourceType = "adjustment"
	SourceTypePayment      LedgerSourceType = "payment"
	SourceTypePaymentFee   LedgerSourceType = "payment_fee"
	SourceTypeCreditGrant  LedgerSourceType = "credit_grant"
	SourceTypeCreditUse    LedgerSourceType = "credit_use"
	SourceTypeRefund       LedgerSourceType = "refund"
	SourceTypeDisputeHold  LedgerSourceType = "dispute_hold"
	SourceTypeDisputeLoss  LedgerSourceType = "dispute_loss"
	SourceTypeDisputeWin   LedgerSourceType = "dispute_win"
)

type LedgerAccountType string

const (
	LedgerAccountTypeAssets    LedgerAccountType = "assets"
	LedgerAccountTypeLiability LedgerAccountType = "liability"
	LedgerAccountTypeIncome    LedgerAccountType = "income"
	LedgerAccountTypeExpense   LedgerAccountType = "expense"
	LedgerAccountTypeEquity    LedgerAccountType = "equity"
)

// LedgerAccount defines a chart-of-accounts entry.
type LedgerAccount struct {
	ID        uuid.UUID         `gorm:"primaryKey" json:"id"`
	OrgID     uuid.UUID         `gorm:"not null;index;uniqueIndex:ux_ledger_accounts_org_code,priority:1" json:"org_id"`
	Code      string            `gorm:"type:text;not null;uniqueIndex:ux_ledger_accounts_org_code,priority:2" json:"code"`
	Type      LedgerAccountType `gorm:"type:text;not null" json:"type"`
	Name      string            `gorm:"type:text;not null" json:"name"`
	CreatedAt time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (LedgerAccount) TableName() string { return "ledger_accounts" }

// LedgerTransaction is the financial event header.
type LedgerTransaction struct {
	ID               uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID            uuid.UUID       `gorm:"not null;index" json:"org_id"`
	Currency         string          `gorm:"type:text;not null" json:"currency"`
	BaseCurrency     *string         `gorm:"type:text" json:"base_currency,omitempty"`
	FXRate           *float64        `gorm:"" json:"fx_rate,omitempty"`
	BaseAmountCents  *int64          `gorm:"" json:"base_amount_cents,omitempty"`
	SourceType       string          `gorm:"type:text;not null;index" json:"source_type"`
	SourceID         uuid.UUID       `gorm:"not null;index" json:"source_id"`
	ReferenceType    *string         `gorm:"type:text" json:"reference_type,omitempty"`
	ReferenceID      *string         `gorm:"type:text" json:"reference_id,omitempty"`
	CustomerID       *uuid.UUID      `gorm:"index" json:"customer_id,omitempty"`
	SubscriptionID   *uuid.UUID      `gorm:"index" json:"subscription_id,omitempty"`
	InvoiceID        *uuid.UUID      `gorm:"index" json:"invoice_id,omitempty"`
	InvoiceItemID    *uuid.UUID      `gorm:"index" json:"invoice_item_id,omitempty"`
	PlanPriceID      *uuid.UUID      `gorm:"index" json:"plan_price_id,omitempty"`
	PlanAmountID     *uuid.UUID      `gorm:"index" json:"plan_amount_id,omitempty"`
	MeterID          *uuid.UUID      `gorm:"index" json:"meter_id,omitempty"`
	RatingResultID   *uuid.UUID      `gorm:"index" json:"rating_result_id,omitempty"`
	UsageAggregateID *uuid.UUID      `gorm:"index" json:"usage_aggregate_id,omitempty"`
	PeriodStart      *time.Time      `gorm:"" json:"period_start,omitempty"`
	PeriodEnd        *time.Time      `gorm:"" json:"period_end,omitempty"`
	OccurredAt       time.Time       `gorm:"not null" json:"occurred_at"`
	PostedAt         time.Time       `gorm:"not null" json:"posted_at"`
	IdempotencyKey   *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata         json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt        time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (LedgerTransaction) TableName() string { return "ledger_transactions" }

// LedgerEntry is a double-entry posting line.
type LedgerEntry struct {
	ID            uuid.UUID       `gorm:"primaryKey" json:"id"`
	TransactionID uuid.UUID       `gorm:"not null;index" json:"transaction_id"`
	OrgID         uuid.UUID       `gorm:"not null;index" json:"org_id"`
	AccountID     *uuid.UUID      `gorm:"index" json:"account_id,omitempty"`
	AccountCode   string          `gorm:"type:text;not null" json:"account_code"`
	EntryType     LedgerEntryType `gorm:"type:text;not null" json:"entry_type"`
	AmountCents   int64           `gorm:"not null" json:"amount_cents"`
	Currency      string          `gorm:"type:text;not null" json:"currency"`
	Description   *string         `gorm:"type:text" json:"description,omitempty"`
	Metadata      json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (LedgerEntry) TableName() string { return "ledger_entries" }
