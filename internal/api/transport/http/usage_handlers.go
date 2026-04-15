package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/ratelimit"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

type ingestUsageRequest struct {
	MeterCode      string  `json:"meter_code"`
	CustomerID     string  `json:"customer_id"`
	SubscriptionID string  `json:"subscription_id,omitempty"`
	Value          float64 `json:"value"`
	RecordedAt     string  `json:"recorded_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

// CreateUsageEvent handles POST /api/v1/usage-events
func (h *Handler) CreateUsageEvent(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	var payload ingestUsageRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	recordedAt, err := parseTime(payload.RecordedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_recorded_at"})
		return
	}
	meterCode := strings.TrimSpace(payload.MeterCode)
	subscriptionID := strings.TrimSpace(payload.SubscriptionID)
	if subscriptionID != "" {
		if parsed, parseErr := uuid.Parse(subscriptionID); parseErr != nil || parsed == uuid.Nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_subscription_id"})
			return
		}
	}
	if h.rateLimiter != nil {
		release, decision, err := h.rateLimiter.AcquireUsageConcurrency(ctx, ratelimit.UsageEventsKey{
			OrgID:      orgID.String(),
			CustomerID: strings.TrimSpace(payload.CustomerID),
			MeterCode:  meterCode,
		})
		if err == nil {
			setRateLimitHeaders(c, decision)
			if !decision.Allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limit_exceeded"})
				return
			}
			defer release()
		}
		decision, err = h.rateLimiter.AllowUsageEvents(ctx, ratelimit.UsageEventsKey{
			OrgID:          orgID.String(),
			CustomerID:     strings.TrimSpace(payload.CustomerID),
			SubscriptionID: subscriptionID,
			MeterCode:      meterCode,
		})
		if err == nil {
			setRateLimitHeaders(c, decision)
			if !decision.Allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limit_exceeded"})
				return
			}
		}
	}
	resp, err := h.usage.IngestUsage(ctx, usagedomain.IngestUsageRequest{
		MeterCode:      meterCode,
		CustomerID:     strings.TrimSpace(payload.CustomerID),
		Value:          payload.Value,
		RecordedAt:     recordedAt,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func setRateLimitHeaders(c *gin.Context, decision ratelimit.Decision) {
	if decision.LimitDisabled || decision.Limit <= 0 {
		return
	}
	c.Header("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
	c.Header("X-RateLimit-Reset", strconv.Itoa(decision.ResetSeconds))
	if decision.Scope != "" {
		c.Header("X-RateLimit-Scope", decision.Scope)
	}
	if decision.Reason != "" {
		c.Header("X-RateLimit-Reason", decision.Reason)
	}
	if !decision.Allowed && decision.ResetSeconds > 0 {
		c.Header("Retry-After", strconv.Itoa(decision.ResetSeconds))
	}
}

// ListUsageEvents handles GET /api/v1/usage-events
func (h *Handler) ListUsageEvents(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	recordedFrom, err := parseTimePtr(c.Query("recorded_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_recorded_from"})
		return
	}
	recordedTo, err := parseTimePtr(c.Query("recorded_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_recorded_to"})
		return
	}
	resp, err := h.usage.ListUsage(ctx, usagedomain.ListUsageRequest{
		PageToken:    c.Query("page_token"),
		PageSize:     parseInt32(c.Query("page_size")),
		MeterID:      c.Query("meter_id"),
		CustomerID:   c.Query("customer_id"),
		Status:       c.Query("status"),
		RecordedFrom: recordedFrom,
		RecordedTo:   recordedTo,
	})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func writeUsageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usagedomain.ErrInvalidOrganization),
		errors.Is(err, usagedomain.ErrInvalidID),
		errors.Is(err, usagedomain.ErrInvalidCode),
		errors.Is(err, usagedomain.ErrInvalidName),
		errors.Is(err, usagedomain.ErrInvalidAggregation),
		errors.Is(err, usagedomain.ErrInvalidUnit),
		errors.Is(err, usagedomain.ErrInvalidMeter),
		errors.Is(err, usagedomain.ErrInvalidCustomer),
		errors.Is(err, usagedomain.ErrInvalidValue),
		errors.Is(err, usagedomain.ErrInvalidRecordedAt),
		errors.Is(err, usagedomain.ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, usagedomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
