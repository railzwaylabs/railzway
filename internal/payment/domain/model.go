package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

type EventRecord struct {
	ID              snowflake.ID   `json:"id" gorm:"primaryKey"`
	OrgID           snowflake.ID   `json:"org_id" gorm:"not null;index"`
	Provider        string         `json:"provider" gorm:"type:text;not null"`
	ProviderEventID string         `json:"provider_event_id" gorm:"type:text;not null"`
	EventType       string         `json:"event_type" gorm:"type:text;not null"`
	CustomerID      snowflake.ID   `json:"customer_id" gorm:"not null;index"`
	Payload         datatypes.JSON `json:"payload" gorm:"type:jsonb;not null"`
	ReceivedAt      time.Time      `json:"received_at" gorm:"not null"`
	ProcessedAt     *time.Time     `json:"processed_at"`
}

func (EventRecord) TableName() string { return "payment_events" }

const (
	EventTypePaymentSucceeded      = "payment_succeeded"
	EventTypePaymentFailed         = "payment_failed"
	EventTypeRefunded              = "refunded"
	EventTypeCheckoutSessionCompleted = "checkout_session_completed"
)

// PaymentEvent is the canonical payment event parsed by adapters.
type PaymentEvent struct {
	Provider            string
	ProviderEventID     string
	ProviderPaymentID   string
	ProviderPaymentType string
	Type                string
	OrgID               snowflake.ID
	CustomerID          snowflake.ID
	Amount              int64
	Currency            string
	OccurredAt          time.Time
	RawPayload          []byte
	InvoiceID           *snowflake.ID
}

// AvailabilityRules defines when a payment method is available
type AvailabilityRules struct {
	Countries  []string `json:"countries"`            // ["ID", "PH"] or ["*"] for all
	Currencies []string `json:"currencies,omitempty"` // ["IDR", "PHP"]
	MinAmount  *int64   `json:"min_amount,omitempty"` // Minimum transaction amount
	MaxAmount  *int64   `json:"max_amount,omitempty"` // Maximum transaction amount
}

// PaymentMethodConfig defines available payment methods and routing rules
type PaymentMethodConfig struct {
	ID                 snowflake.ID      `json:"id" gorm:"primaryKey"`
	OrgID              snowflake.ID      `json:"org_id" gorm:"not null;index"`
	MethodType         string            `json:"method_type" gorm:"type:varchar(50);not null"`         // 'card', 'virtual_account', 'ewallet'
	MethodName         string            `json:"method_name" gorm:"type:varchar(100);not null"`        // 'card_global', 'va_bca', 'gopay'
	AvailabilityRules  datatypes.JSON    `json:"availability_rules" gorm:"type:jsonb;not null"`        // JSON rules
	Provider           string            `json:"provider" gorm:"type:varchar(50);not null"`            // 'xendit', 'stripe'
	ProviderMethodType string            `json:"provider_method_type" gorm:"type:varchar(50)"`         // Provider-specific type
	DisplayName        string            `json:"display_name" gorm:"type:varchar(100);not null"`       // User-facing name
	Description        string            `json:"description" gorm:"type:text"`                         // User-facing description
	IconURL            string            `json:"icon_url" gorm:"type:varchar(255)"`                    // Icon URL
	Priority           int               `json:"priority" gorm:"default:0"`                            // Display priority
	IsActive           bool              `json:"is_active" gorm:"default:true"`                        // Enabled/disabled
	CreatedAt          time.Time         `json:"created_at" gorm:"not null"`
	UpdatedAt          time.Time         `json:"updated_at" gorm:"not null"`
	PublicKey          string            `json:"public_key,omitempty" gorm:"-"`                        // Transient field from provider config
}

func (PaymentMethodConfig) TableName() string { return "payment_method_configs" }

// PaymentMethod represents a tokenized payment method stored for a customer
type PaymentMethod struct {
	ID                       snowflake.ID `json:"id" gorm:"primaryKey"`
	CustomerID               snowflake.ID `json:"customer_id" gorm:"not null;index"`
	Type                     string       `json:"type" gorm:"type:varchar(50);not null"`                  // 'card', 'virtual_account', 'ewallet'
	Provider                 string       `json:"provider" gorm:"type:varchar(50);not null"`              // 'xendit', 'stripe'
	ProviderPaymentMethodID  string       `json:"provider_payment_method_id" gorm:"type:varchar(255);not null"` // Token
	Last4                    string       `json:"last4" gorm:"type:varchar(4)"`                           // Last 4 digits
	Brand                    string       `json:"brand" gorm:"type:varchar(50)"`                          // 'visa', 'bca', 'gopay'
	ExpMonth                 int          `json:"exp_month"`                                              // Expiry month (cards)
	ExpYear                  int          `json:"exp_year"`                                               // Expiry year (cards)
	IsDefault                bool         `json:"is_default" gorm:"default:false"`                        // Default payment method
	CreatedAt                time.Time    `json:"created_at" gorm:"not null"`
	UpdatedAt                time.Time    `json:"updated_at" gorm:"not null"`
}

func (PaymentMethod) TableName() string { return "customer_payment_methods" }

// PaymentMethodDetails is returned by provider adapters (not stored in DB)
type PaymentMethodDetails struct {
	ID       string // Provider payment method ID
	Type     string // 'card', 'ewallet', 'virtual_account'
	Last4    string // Last 4 digits
	Brand    string // 'visa', 'mastercard', 'bca', 'gopay'
	ExpMonth int    // Expiry month (if applicable)
	ExpYear  int    // Expiry year (if applicable)
}




