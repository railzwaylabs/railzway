package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	coupondomain "github.com/railzwaylabs/railzway/internal/coupon/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type createCouponRequest struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	AmountCents    *int64   `json:"amount_cents"`
	Percentage     *float64 `json:"percentage"`
	Duration       string   `json:"duration"`
	DurationMonths *int     `json:"duration_months"`
	Currency       *string  `json:"currency"`
	ValidFrom      string   `json:"valid_from"`
	ValidUntil     string   `json:"valid_until"`
	AutoApply      *bool    `json:"auto_apply"`
	TargetSegment  *string  `json:"target_segment"`
}

type createPromotionCodeRequest struct {
	Code           string `json:"code"`
	CouponID       string `json:"coupon_id"`
	Active         *bool  `json:"active"`
	MaxRedemptions *int   `json:"max_redemptions"`
}

type redeemPromotionCodeRequest struct {
	Code string `json:"code"`
}

type createSegmentRequest struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Scope       string  `json:"scope"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

type updateSegmentRequest struct {
	Name        *string `json:"name"`
	Scope       *string `json:"scope"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createCouponRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	validFrom, err := parseTimePtr(payload.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_coupon"})
		return
	}
	validUntil, err := parseTimePtr(payload.ValidUntil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_coupon"})
		return
	}
	autoApply := false
	if payload.AutoApply != nil {
		autoApply = *payload.AutoApply
	}

	resp, err := h.coupons.CreateCoupon(ctx, coupondomain.CreateCouponRequest{
		Name:           strings.TrimSpace(payload.Name),
		Type:           strings.TrimSpace(payload.Type),
		AmountCents:    payload.AmountCents,
		Percentage:     payload.Percentage,
		Duration:       strings.TrimSpace(payload.Duration),
		DurationMonths: payload.DurationMonths,
		Currency:       payload.Currency,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		AutoApply:      autoApply,
		TargetSegment:  payload.TargetSegment,
	})
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"coupon": resp})
}

func (h *Handler) ListCoupons(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.coupons.ListCoupons(ctx)
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"coupons": resp})
}

func (h *Handler) CreatePromotionCode(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createPromotionCodeRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	couponID, err := uuid.Parse(strings.TrimSpace(payload.CouponID))
	if err != nil || couponID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_coupon"})
		return
	}
	active := true
	if payload.Active != nil {
		active = *payload.Active
	}

	resp, err := h.coupons.CreatePromotionCode(ctx, coupondomain.CreatePromotionCodeRequest{
		Code:           strings.TrimSpace(payload.Code),
		CouponID:       couponID,
		Active:         active,
		MaxRedemptions: payload.MaxRedemptions,
	})
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"promotion_code": resp})
}

func (h *Handler) ListPromotionCodes(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.coupons.ListPromotionCodes(ctx)
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"promotion_codes": resp})
}

func (h *Handler) CreateSegment(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createSegmentRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.coupons.CreateSegment(ctx, coupondomain.CreateSegmentRequest{
		Key:         strings.TrimSpace(payload.Key),
		Name:        strings.TrimSpace(payload.Name),
		Scope:       strings.TrimSpace(payload.Scope),
		Description: payload.Description,
		Active:      payload.Active,
	})
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"segment": resp})
}

func (h *Handler) ListSegments(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.coupons.ListSegments(ctx, coupondomain.ListSegmentsRequest{
		Scope:           strings.TrimSpace(c.Query("scope")),
		IncludeInactive: strings.EqualFold(strings.TrimSpace(c.Query("include_inactive")), "true"),
	})
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"segments": resp})
}

func (h *Handler) UpdateSegment(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload updateSegmentRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.coupons.UpdateSegment(ctx, strings.TrimSpace(c.Param("segment_key")), coupondomain.UpdateSegmentRequest{
		Name:        payload.Name,
		Scope:       payload.Scope,
		Description: payload.Description,
		Active:      payload.Active,
	})
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"segment": resp})
}

func (h *Handler) RedeemPromotionCode(c *gin.Context) {
	if h.coupons == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "coupon_service_not_configured"})
		return
	}
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	subscriptionID, err := uuid.Parse(strings.TrimSpace(c.Param("subscription_id")))
	if err != nil || subscriptionID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_subscription"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload redeemPromotionCodeRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.coupons.RedeemPromotionCode(ctx, payload.Code, subscriptionID)
	if err != nil {
		writeCouponError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"coupon": resp})
}

func writeCouponError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, coupondomain.ErrInvalidCoupon):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_coupon"})
	case errors.Is(err, coupondomain.ErrInvalidPromotionCode):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_promotion_code"})
	case errors.Is(err, coupondomain.ErrPromotionCodeInactive):
		c.JSON(http.StatusConflict, gin.H{"error": "promotion_code_inactive"})
	case errors.Is(err, coupondomain.ErrPromotionCodeUsed):
		c.JSON(http.StatusConflict, gin.H{"error": "promotion_code_max_redemptions_reached"})
	case errors.Is(err, coupondomain.ErrInvalidSegment):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_segment"})
	case errors.Is(err, coupondomain.ErrSegmentExists):
		c.JSON(http.StatusConflict, gin.H{"error": "segment_exists"})
	case errors.Is(err, coupondomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
