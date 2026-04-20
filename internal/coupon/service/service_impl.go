package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/coupon/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"gorm.io/gorm"
)

type service struct {
	db    *gorm.DB
	repo  domain.Repository
	audit *auditlog.Service
}

func NewService(db *gorm.DB, repo domain.Repository, audit *auditlog.Service) domain.Service {
	return &service{
		db:    db,
		repo:  repo,
		audit: audit,
	}
}

func (s *service) CreateCoupon(ctx context.Context, req domain.CreateCouponRequest) (*domain.Coupon, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	if err := validateCouponRequest(req); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	couponType := strings.TrimSpace(strings.ToUpper(req.Type))
	duration := strings.TrimSpace(strings.ToUpper(req.Duration))
	var currency *string
	if req.Currency != nil && strings.TrimSpace(*req.Currency) != "" {
		value := strings.TrimSpace(strings.ToUpper(*req.Currency))
		currency = &value
	}

	coupon := domain.Coupon{
		ID:             uuid.New(),
		OrgID:          orgID,
		Name:           name,
		Type:           couponType,
		AmountCents:    req.AmountCents,
		Percentage:     req.Percentage,
		Duration:       duration,
		DurationMonths: req.DurationMonths,
		Currency:       currency,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.CreateCoupon(ctx, coupon); err != nil {
		return nil, err
	}

	return &coupon, nil
}

func (s *service) GetCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s.repo.GetCoupon(ctx, orgID, id)
}

func (s *service) ListCoupons(ctx context.Context) ([]domain.Coupon, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s.repo.ListCoupons(ctx, orgID)
}

func (s *service) CreatePromotionCode(ctx context.Context, req domain.CreatePromotionCodeRequest) (*domain.PromotionCode, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	code := normalizePromotionCode(req.Code)
	if code == "" || req.CouponID == uuid.Nil {
		return nil, domain.ErrInvalidPromotionCode
	}
	coupon, err := s.repo.GetCoupon(ctx, orgID, req.CouponID)
	if err != nil {
		return nil, err
	}
	if coupon == nil {
		return nil, domain.ErrInvalidCoupon
	}

	promo := domain.PromotionCode{
		ID:             uuid.New(),
		OrgID:          orgID,
		CouponID:       req.CouponID,
		Code:           code,
		Active:         req.Active,
		MaxRedemptions: req.MaxRedemptions,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.CreatePromotionCode(ctx, promo); err != nil {
		return nil, err
	}

	return &promo, nil
}

func (s *service) GetPromotionCode(ctx context.Context, code string) (*domain.PromotionCode, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s.repo.GetPromotionCode(ctx, orgID, normalizePromotionCode(code))
}

func (s *service) ApplyCouponToSubscription(ctx context.Context, subID uuid.UUID, couponID uuid.UUID) error {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return domain.ErrNotFound
	}
	if subID == uuid.Nil || couponID == uuid.Nil {
		return domain.ErrInvalidCoupon
	}
	coupon, err := s.repo.GetCoupon(ctx, orgID, couponID)
	if err != nil {
		return err
	}
	if coupon == nil {
		return domain.ErrInvalidCoupon
	}

	subCoupon := domain.SubscriptionCoupon{
		ID:             uuid.New(),
		OrgID:          orgID,
		SubscriptionID: subID,
		CouponID:       couponID,
		AppliedAt:      time.Now().UTC(),
	}

	return s.repo.ApplyCoupon(ctx, subCoupon)
}

func (s *service) GetAttachedCoupon(ctx context.Context, subID uuid.UUID) (*domain.Coupon, error) {
	details, err := s.GetAttachedCouponDetails(ctx, subID)
	if err != nil || details == nil {
		return nil, err
	}
	return &details.Coupon, nil
}

func (s *service) GetAttachedCouponDetails(ctx context.Context, subID uuid.UUID) (*domain.AttachedCouponDetails, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}

	subCoupon, err := s.repo.GetSubscriptionCoupon(ctx, orgID, subID)
	if err != nil {
		return nil, err
	}
	if subCoupon == nil {
		return nil, nil
	}

	coupon, err := s.repo.GetCoupon(ctx, orgID, subCoupon.CouponID)
	if err != nil {
		return nil, err
	}
	if coupon == nil {
		return nil, nil
	}
	return &domain.AttachedCouponDetails{
		Coupon:    *coupon,
		AppliedAt: subCoupon.AppliedAt,
	}, nil
}

func (s *service) RedeemPromotionCode(ctx context.Context, code string, subID uuid.UUID) (*domain.Coupon, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	normalized := normalizePromotionCode(code)
	if normalized == "" || subID == uuid.Nil {
		return nil, domain.ErrInvalidPromotionCode
	}

	var coupon *domain.Coupon
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		promo, err := repo.GetPromotionCode(ctx, orgID, normalized)
		if err != nil {
			return err
		}
		if promo == nil {
			return domain.ErrNotFound
		}
		if !promo.Active {
			return domain.ErrPromotionCodeInactive
		}
		if promo.MaxRedemptions != nil && promo.RedemptionCount >= *promo.MaxRedemptions {
			return domain.ErrPromotionCodeUsed
		}
		c, err := repo.GetCoupon(ctx, orgID, promo.CouponID)
		if err != nil {
			return err
		}
		if c == nil {
			return domain.ErrInvalidCoupon
		}
		now := time.Now().UTC()
		if err := repo.ApplyCoupon(ctx, domain.SubscriptionCoupon{
			ID:             uuid.New(),
			OrgID:          orgID,
			SubscriptionID: subID,
			CouponID:       c.ID,
			AppliedAt:      now,
		}); err != nil {
			return err
		}
		if err := repo.IncrementRedemptionCount(ctx, orgID, promo.ID); err != nil {
			return err
		}
		coupon = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return coupon, nil
}

func normalizePromotionCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func validateCouponRequest(req domain.CreateCouponRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return domain.ErrInvalidCoupon
	}
	couponType := strings.TrimSpace(strings.ToUpper(req.Type))
	duration := strings.TrimSpace(strings.ToUpper(req.Duration))
	switch couponType {
	case domain.CouponTypeFixed:
		if req.AmountCents == nil || *req.AmountCents <= 0 {
			return domain.ErrInvalidCoupon
		}
		if req.Currency == nil || len(strings.TrimSpace(*req.Currency)) != 3 {
			return domain.ErrInvalidCoupon
		}
	case domain.CouponTypePercent:
		if req.Percentage == nil || *req.Percentage <= 0 || *req.Percentage > 100 {
			return domain.ErrInvalidCoupon
		}
	default:
		return domain.ErrInvalidCoupon
	}
	switch duration {
	case domain.CouponDurationOnce, domain.CouponDurationForever:
		return nil
	case domain.CouponDurationRepeating:
		if req.DurationMonths == nil || *req.DurationMonths <= 0 {
			return domain.ErrInvalidCoupon
		}
		return nil
	default:
		return domain.ErrInvalidCoupon
	}
}
