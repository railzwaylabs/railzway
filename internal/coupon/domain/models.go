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

	CouponApplicationSourceSubscription = "subscription_coupon"
	CouponApplicationSourceAutoApply    = "auto_apply"

	SegmentScopeAny          = "any"
	SegmentScopeCustomer     = "customer"
	SegmentScopeSubscription = "subscription"
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
	ValidFrom      *time.Time      `gorm:"" json:"valid_from,omitempty"`
	ValidUntil     *time.Time      `gorm:"" json:"valid_until,omitempty"`
	AutoApply      bool            `gorm:"not null;default:false" json:"auto_apply"`
	TargetSegment  *string         `gorm:"type:text" json:"target_segment,omitempty"`
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

type Segment struct {
	ID          uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID       uuid.UUID       `gorm:"not null;index" json:"org_id"`
	Key         string          `gorm:"type:text;not null" json:"key"`
	Name        string          `gorm:"type:text;not null" json:"name"`
	Scope       string          `gorm:"type:text;not null" json:"scope"`
	Description *string         `gorm:"type:text" json:"description,omitempty"`
	Active      bool            `gorm:"not null;default:true" json:"active"`
	Metadata    json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Segment) TableName() string { return "billing_segments" }
