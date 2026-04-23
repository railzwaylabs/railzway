package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/coupon/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"gorm.io/gorm"
)

var segmentKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

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
	var targetSegment *string
	if req.TargetSegment != nil && strings.TrimSpace(*req.TargetSegment) != "" {
		value := normalizeSegmentKey(*req.TargetSegment)
		if value == "" {
			return nil, domain.ErrInvalidSegment
		}
		if err := s.ensureDefaultSegments(ctx, orgID); err != nil {
			return nil, err
		}
		segment, err := s.repo.GetSegmentByKey(ctx, orgID, value)
		if err != nil {
			return nil, err
		}
		if segment == nil || !segment.Active {
			return nil, domain.ErrInvalidSegment
		}
		targetSegment = &value
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
		ValidFrom:      normalizeTimePtr(req.ValidFrom),
		ValidUntil:     normalizeTimePtr(req.ValidUntil),
		AutoApply:      req.AutoApply,
		TargetSegment:  targetSegment,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.CreateCoupon(ctx, coupon); err != nil {
		return nil, err
	}

	return &coupon, nil
}

func (s *service) CreateSegment(ctx context.Context, req domain.CreateSegmentRequest) (*domain.Segment, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	key := normalizeSegmentKey(req.Key)
	name := strings.TrimSpace(req.Name)
	scope := normalizeSegmentScopeOrDefault(req.Scope)
	if key == "" || name == "" || scope == "" {
		return nil, domain.ErrInvalidSegment
	}
	existing, err := s.repo.GetSegmentByKey(ctx, orgID, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrSegmentExists
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}
	description := normalizeOptionalString(req.Description)
	now := time.Now().UTC()
	segment := domain.Segment{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         key,
		Name:        name,
		Scope:       scope,
		Description: description,
		Active:      active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateSegment(ctx, segment); err != nil {
		return nil, err
	}
	return &segment, nil
}

func (s *service) UpdateSegment(ctx context.Context, key string, req domain.UpdateSegmentRequest) (*domain.Segment, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	normalizedKey := normalizeSegmentKey(key)
	if normalizedKey == "" {
		return nil, domain.ErrInvalidSegment
	}
	if err := s.ensureDefaultSegments(ctx, orgID); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetSegmentByKey(ctx, orgID, normalizedKey)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrNotFound
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidSegment
		}
		updates["name"] = name
	}
	if req.Scope != nil {
		scope := normalizeSegmentScope(*req.Scope)
		if scope == "" {
			return nil, domain.ErrInvalidSegment
		}
		updates["scope"] = scope
	}
	if req.Description != nil {
		updates["description"] = normalizeOptionalString(req.Description)
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if err := s.repo.UpdateSegment(ctx, orgID, normalizedKey, updates); err != nil {
		return nil, err
	}
	return s.repo.GetSegmentByKey(ctx, orgID, normalizedKey)
}

func (s *service) ListSegments(ctx context.Context, req domain.ListSegmentsRequest) ([]domain.Segment, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	if err := s.ensureDefaultSegments(ctx, orgID); err != nil {
		return nil, err
	}
	scope := normalizeSegmentScope(req.Scope)
	if strings.TrimSpace(req.Scope) != "" && scope == "" {
		return nil, domain.ErrInvalidSegment
	}
	return s.repo.ListSegments(ctx, orgID, scope, req.IncludeInactive)
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

func (s *service) ListAutoApplyCoupons(ctx context.Context, periodStart, periodEnd time.Time) ([]domain.Coupon, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s.repo.ListAutoApplyCoupons(ctx, orgID, periodStart.UTC(), periodEnd.UTC())
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

func (s *service) ListPromotionCodes(ctx context.Context) ([]domain.PromotionCode, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s.repo.ListPromotionCodes(ctx, orgID)
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

func normalizeSegmentKey(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	if !segmentKeyPattern.MatchString(normalized) {
		return ""
	}
	return normalized
}

func normalizeSegmentScope(scope string) string {
	normalized := strings.ToLower(strings.TrimSpace(scope))
	switch normalized {
	case domain.SegmentScopeAny, domain.SegmentScopeCustomer, domain.SegmentScopeSubscription:
		return normalized
	default:
		return ""
	}
}

func normalizeSegmentScopeOrDefault(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return domain.SegmentScopeCustomer
	}
	return normalizeSegmentScope(scope)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func (s *service) ensureDefaultSegments(ctx context.Context, orgID uuid.UUID) error {
	now := time.Now().UTC()
	segments := []domain.Segment{
		defaultSegment(orgID, "startup", "Startup", domain.SegmentScopeCustomer, "Early-stage or venture-backed customer", now),
		defaultSegment(orgID, "enterprise", "Enterprise", domain.SegmentScopeCustomer, "Large customer with enterprise billing needs", now),
		defaultSegment(orgID, "education", "Education", domain.SegmentScopeCustomer, "School, university, or education program", now),
		defaultSegment(orgID, "partner", "Partner", domain.SegmentScopeCustomer, "Partner or reseller customer", now),
		defaultSegment(orgID, "internal", "Internal", domain.SegmentScopeAny, "Internal, test, or demo usage", now),
		defaultSegment(orgID, "self_serve", "Self Serve", domain.SegmentScopeSubscription, "Subscription created through self-serve flow", now),
		defaultSegment(orgID, "sales_led", "Sales Led", domain.SegmentScopeSubscription, "Subscription created through a sales-led motion", now),
		defaultSegment(orgID, "annual_contract", "Annual Contract", domain.SegmentScopeSubscription, "Subscription tied to an annual agreement", now),
		defaultSegment(orgID, "trial_converted", "Trial Converted", domain.SegmentScopeSubscription, "Subscription converted from trial", now),
		defaultSegment(orgID, "reactivated", "Reactivated", domain.SegmentScopeSubscription, "Subscription reactivated after cancellation", now),
		defaultSegment(orgID, "loyal_customer", "Loyal Customer", domain.SegmentScopeCustomer, "Long-tenured customer", now),
		defaultSegment(orgID, "high_spend", "High Spend", domain.SegmentScopeCustomer, "Customer with high invoice spend", now),
		defaultSegment(orgID, "committed_spend", "Committed Spend", domain.SegmentScopeCustomer, "Customer with committed spend terms", now),
	}
	return s.repo.EnsureSegments(ctx, segments)
}

func defaultSegment(orgID uuid.UUID, key, name, scope, description string, now time.Time) domain.Segment {
	return domain.Segment{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         key,
		Name:        name,
		Scope:       scope,
		Description: &description,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func normalizeTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func validateCouponRequest(req domain.CreateCouponRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return domain.ErrInvalidCoupon
	}
	couponType := strings.TrimSpace(strings.ToUpper(req.Type))
	duration := strings.TrimSpace(strings.ToUpper(req.Duration))
	if req.ValidFrom != nil && req.ValidUntil != nil && !req.ValidUntil.After(*req.ValidFrom) {
		return domain.ErrInvalidCoupon
	}
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
