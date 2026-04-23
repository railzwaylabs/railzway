package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	CreateCoupon(ctx context.Context, req CreateCouponRequest) (*Coupon, error)
	GetCoupon(ctx context.Context, id uuid.UUID) (*Coupon, error)
	ListCoupons(ctx context.Context) ([]Coupon, error)
	ListAutoApplyCoupons(ctx context.Context, periodStart, periodEnd time.Time) ([]Coupon, error)

	CreateSegment(ctx context.Context, req CreateSegmentRequest) (*Segment, error)
	UpdateSegment(ctx context.Context, key string, req UpdateSegmentRequest) (*Segment, error)
	ListSegments(ctx context.Context, req ListSegmentsRequest) ([]Segment, error)

	CreatePromotionCode(ctx context.Context, req CreatePromotionCodeRequest) (*PromotionCode, error)
	GetPromotionCode(ctx context.Context, code string) (*PromotionCode, error)
	ListPromotionCodes(ctx context.Context) ([]PromotionCode, error)

	ApplyCouponToSubscription(ctx context.Context, subID uuid.UUID, couponID uuid.UUID) error
	GetAttachedCoupon(ctx context.Context, subID uuid.UUID) (*Coupon, error)
	GetAttachedCouponDetails(ctx context.Context, subID uuid.UUID) (*AttachedCouponDetails, error)
	RedeemPromotionCode(ctx context.Context, code string, subID uuid.UUID) (*Coupon, error)
}

type CreateCouponRequest struct {
	Name           string
	Type           string
	AmountCents    *int64
	Percentage     *float64
	Duration       string
	DurationMonths *int
	Currency       *string
	ValidFrom      *time.Time
	ValidUntil     *time.Time
	AutoApply      bool
	TargetSegment  *string
}

type CreateSegmentRequest struct {
	Key         string
	Name        string
	Scope       string
	Description *string
	Active      *bool
}

type UpdateSegmentRequest struct {
	Name        *string
	Scope       *string
	Description *string
	Active      *bool
}

type ListSegmentsRequest struct {
	Scope           string
	IncludeInactive bool
}

type CreatePromotionCodeRequest struct {
	Code           string
	CouponID       uuid.UUID
	Active         bool
	MaxRedemptions *int
}

type AttachedCouponDetails struct {
	Coupon    Coupon
	AppliedAt time.Time
}

var (
	ErrInvalidCoupon         = errors.New("invalid_coupon")
	ErrInvalidPromotionCode  = errors.New("invalid_promotion_code")
	ErrPromotionCodeUsed     = errors.New("promotion_code_max_redemptions_reached")
	ErrPromotionCodeInactive = errors.New("promotion_code_inactive")
	ErrInvalidSegment        = errors.New("invalid_segment")
	ErrSegmentExists         = errors.New("segment_exists")
	ErrNotFound              = errors.New("not_found")
)
