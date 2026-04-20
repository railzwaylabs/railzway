package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	CouponTypePercent = "PERCENT"
	CouponTypeFixed   = "FIXED"

	CouponDurationOnce      = "ONCE"
	CouponDurationForever   = "FOREVER"
	CouponDurationRepeating = "REPEATING"
)

type Coupon struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"not null;index" json:"org_id"`
	Name           string          `gorm:"type:text;not null" json:"name"`
	Type           string          `gorm:"type:text;not null" json:"type"`
	AmountCents    *int64          `gorm:"" json:"amount_cents,omitempty"`
	Percentage     *float64        `gorm:"" json:"percentage,omitempty"`
	Duration       string          `gorm:"type:text;not null" json:"duration"`
	DurationMonths *int            `gorm:"" json:"duration_months,omitempty"`
	Currency       *string         `gorm:"type:text" json:"currency,omitempty"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Coupon) TableName() string { return "coupons" }

type PromotionCode struct {
	ID              uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID           uuid.UUID       `gorm:"not null;index" json:"org_id"`
	CouponID        uuid.UUID       `gorm:"not null;index" json:"coupon_id"`
	Code            string          `gorm:"type:text;not null" json:"code"`
	Active          bool            `gorm:"not null" json:"active"`
	MaxRedemptions  *int            `gorm:"" json:"max_redemptions,omitempty"`
	RedemptionCount int             `gorm:"not null" json:"redemption_count"`
	Metadata        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PromotionCode) TableName() string { return "promotion_codes" }

type SubscriptionCoupon struct {
	ID             uuid.UUID `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID `gorm:"not null;index" json:"org_id"`
	SubscriptionID uuid.UUID `gorm:"not null;index" json:"subscription_id"`
	CouponID       uuid.UUID `gorm:"not null;index" json:"coupon_id"`
	AppliedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"applied_at"`
}

func (SubscriptionCoupon) TableName() string { return "subscription_coupons" }
