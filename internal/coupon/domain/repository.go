package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateCoupon(ctx context.Context, coupon Coupon) error
	GetCoupon(ctx context.Context, orgID, id uuid.UUID) (*Coupon, error)
	ListCoupons(ctx context.Context, orgID uuid.UUID) ([]Coupon, error)
	ListAutoApplyCoupons(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]Coupon, error)

	CreateSegment(ctx context.Context, segment Segment) error
	EnsureSegments(ctx context.Context, segments []Segment) error
	GetSegmentByKey(ctx context.Context, orgID uuid.UUID, key string) (*Segment, error)
	ListSegments(ctx context.Context, orgID uuid.UUID, scope string, includeInactive bool) ([]Segment, error)
	UpdateSegment(ctx context.Context, orgID uuid.UUID, key string, updates map[string]interface{}) error

	CreatePromotionCode(ctx context.Context, promo PromotionCode) error
	GetPromotionCode(ctx context.Context, orgID uuid.UUID, code string) (*PromotionCode, error)
	ListPromotionCodes(ctx context.Context, orgID uuid.UUID) ([]PromotionCode, error)
	IncrementRedemptionCount(ctx context.Context, orgID, id uuid.UUID) error

	ApplyCoupon(ctx context.Context, subCoupon SubscriptionCoupon) error
	GetSubscriptionCoupon(ctx context.Context, orgID, subID uuid.UUID) (*SubscriptionCoupon, error)
}
