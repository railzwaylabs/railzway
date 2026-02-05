package domain

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrCheckoutSessionNotFound = errors.New("checkout session not found")
	ErrInvalidCheckoutSession  = errors.New("invalid checkout session")
)

type CheckoutSessionStatus string
type PaymentStatus string

const (
	CheckoutSessionStatusOpen     CheckoutSessionStatus = "open"
	CheckoutSessionStatusComplete CheckoutSessionStatus = "complete"
	CheckoutSessionStatusExpired  CheckoutSessionStatus = "expired"

	PaymentStatusUnpaid PaymentStatus = "unpaid"
	PaymentStatusPaid   PaymentStatus = "paid"
)

type CheckoutSession struct {
	ID                snowflake.ID          `json:"id" gorm:"primaryKey;autoIncrement:false"`
	OrgID             snowflake.ID          `json:"org_id" gorm:"not null"`
	CustomerID        *snowflake.ID         `json:"customer_id" gorm:"type:bigint"`
	Provider          string                `json:"provider" gorm:"type:varchar(50)"`
	Status            CheckoutSessionStatus `json:"status" gorm:"type:varchar(20)"`
	PaymentStatus     PaymentStatus         `json:"payment_status" gorm:"type:varchar(20)"`
	LineItems         datatypes.JSON        `json:"line_items" gorm:"type:jsonb"`
	AmountTotal       int64                 `json:"amount_total" gorm:"not null"`
	Currency          string                `json:"currency" gorm:"type:varchar(3)"`
	SuccessURL        string                `json:"success_url" gorm:"type:text"`
	CancelURL         string                `json:"cancel_url" gorm:"type:text"`
	PaymentIntentID   string                `json:"payment_intent_id" gorm:"type:varchar(255)"`
	SubscriptionID    string                `json:"subscription_id" gorm:"type:varchar(255)"` // Now stored, not just DTO
	ProviderSessionID string                `json:"provider_session_id" gorm:"type:varchar(255)"`
	ClientReferenceID string                `json:"client_reference_id" gorm:"type:varchar(255)"`
	Metadata          datatypes.JSONMap     `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	ExpiresAt         *time.Time            `json:"expires_at"`
	CompletedAt       *time.Time            `json:"completed_at"`
	ExpiredAt         *time.Time            `json:"expired_at"`
	CreatedAt         time.Time             `json:"created_at" gorm:"not null"`
	UpdatedAt         time.Time             `json:"updated_at" gorm:"not null"`

	// DTO Helper fields (not stored)
	URL string `json:"url,omitempty" gorm:"-"`
}

// ProviderCheckoutSession is returned by the adapter
type ProviderCheckoutSession struct {
	ID        string                `json:"id"`
	Provider  string                `json:"provider"`
	URL       string                `json:"url"`
	Status          CheckoutSessionStatus `json:"status"`
	ExpiresAt       time.Time             `json:"expires_at"`
	PaymentMethodID string                `json:"payment_method_id,omitempty"`
	PaymentIntentID string                `json:"payment_intent_id,omitempty"`
}

// LineItemInput represents a line item for checkout
// PriceID is the railzway-oss price ID (not provider-specific)
type LineItemInput struct {
	PriceID  string `json:"price" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// LineItem represents a line item in the response
type LineItem struct {
	PriceID     string `json:"price_id"`
	Quantity    int    `json:"quantity"`
	UnitAmount  int64  `json:"unit_amount"`
	AmountTotal int64  `json:"amount_total"`
}

type CheckoutSessionInput struct {
	OrgID               snowflake.ID      `json:"org_id"`
	Provider            string            `json:"provider"` // 'stripe', 'xendit' (defaults to org default if empty)
	CustomerID          snowflake.ID      `json:"customer" binding:"required"`
	ProviderCustomerID  string            `json:"provider_customer_id"` // e.g. cus_...
	Currency            string            `json:"currency" binding:"required,len=3"` // ISO currency code (USD, IDR, etc)
	
	// Line Items (REQUIRED) - uses railzway-oss price_id
	LineItems           []LineItemInput   `json:"line_items" binding:"required,min=1,dive"`
	
	// Calculated fields (set by service after price lookup)
	Amount              int64             `json:"-"` // Total amount in cents (calculated from line_items)
	
	// Common fields
	SuccessURL          string            `json:"success_url" binding:"required"`
	CancelURL           string            `json:"cancel_url" binding:"required"`
	ClientReferenceID   string            `json:"client_reference_id,omitempty"`
	Metadata            map[string]string `json:"metadata"`
	AllowPromotionCodes bool              `json:"allow_promotion_codes"`
}

type CheckoutSessionRepository interface {
	Insert(ctx context.Context, db *gorm.DB, session *CheckoutSession) error
	Update(ctx context.Context, db *gorm.DB, session *CheckoutSession) error
	FindByID(ctx context.Context, db *gorm.DB, id snowflake.ID) (*CheckoutSession, error)
	FindByProviderSessionID(ctx context.Context, db *gorm.DB, provider, providerSessionID string) (*CheckoutSession, error)
	FindByAnyProviderSessionID(ctx context.Context, db *gorm.DB, providerSessionID string) (*CheckoutSession, error)
}
