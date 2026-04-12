package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
)

func (h *Handler) ListRatingResults(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	windowStartFrom, err := parseTimePtr(c.Query("window_start_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_window_start_from"})
		return
	}
	windowStartTo, err := parseTimePtr(c.Query("window_start_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_window_start_to"})
		return
	}

	resp, err := h.rating.ListRatingResults(ctx, ratingdomain.ListRatingResultsRequest{
		PageToken:       c.Query("page_token"),
		PageSize:        parseInt32(c.Query("page_size")),
		CustomerID:      c.Query("customer_id"),
		SubscriptionID:  c.Query("subscription_id"),
		PlanPriceID:     c.Query("plan_price_id"),
		MeterID:         c.Query("meter_id"),
		UsageEventID:    c.Query("usage_event_id"),
		WindowStartFrom: windowStartFrom,
		WindowStartTo:   windowStartTo,
	})
	if err != nil {
		writeRatingError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListUsageAggregates(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	periodStartFrom, err := parseTimePtr(c.Query("period_start_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_period_start_from"})
		return
	}
	periodStartTo, err := parseTimePtr(c.Query("period_start_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_period_start_to"})
		return
	}

	resp, err := h.rating.ListUsageAggregates(ctx, ratingdomain.ListUsageAggregatesRequest{
		PageToken:       c.Query("page_token"),
		PageSize:        parseInt32(c.Query("page_size")),
		CustomerID:      c.Query("customer_id"),
		SubscriptionID:  c.Query("subscription_id"),
		PlanPriceID:     c.Query("plan_price_id"),
		MeterID:         c.Query("meter_id"),
		PeriodStartFrom: periodStartFrom,
		PeriodStartTo:   periodStartTo,
	})
	if err != nil {
		writeRatingError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RateUsageEvent(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	usageEventID := strings.TrimSpace(c.Param("usage_event_id"))
	resp, err := h.rating.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: usageEventID})
	if err != nil {
		writeRatingError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
