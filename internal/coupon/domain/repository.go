package domain

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateCoupon(ctx context.Context, coupon Coupon) error
	GetCoupon(ctx context.Context, orgID, id uuid.UUID) (*Coupon, error)
	ListCoupons(ctx context.Context, orgID uuid.UUID) ([]Coupon, error)

	CreatePromotionCode(ctx context.Context, promo PromotionCode) error
	GetPromotionCode(ctx context.Context, orgID uuid.UUID, code string) (*PromotionCode, error)
	IncrementRedemptionCount(ctx context.Context, orgID, id uuid.UUID) error

	ApplyCoupon(ctx context.Context, subCoupon SubscriptionCoupon) error
	GetSubscriptionCoupon(ctx context.Context, orgID, subID uuid.UUID) (*SubscriptionCoupon, error)
}
