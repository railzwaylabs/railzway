package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
)

type createSubscriptionRequest struct {
	CustomerID         string                                           `json:"customer_id"`
	PlanID             string                                           `json:"plan_id"`
	Currency           string                                           `json:"currency"`
	StartAt            *string                                          `json:"start_at"`
	CurrentPeriodStart string                                           `json:"current_period_start"`
	CurrentPeriodEnd   string                                           `json:"current_period_end"`
	TrialEnd           *string                                          `json:"trial_end"`
	CancelAt           *string                                          `json:"cancel_at"`
	Status             string                                           `json:"status"`
	IdempotencyKey     string                                           `json:"idempotency_key"`
	Items              []subscriptiondomain.CreateSubscriptionItemInput `json:"items"`
}

// CreateSubscription handles POST /api/v1/subscriptions
func (h *Handler) CreateSubscription(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	var payload createSubscriptionRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	startAt, err := parseTimePtrValue(payload.StartAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_start_at"})
		return
	}
	trialEnd, err := parseTimePtrValue(payload.TrialEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_trial_end"})
		return
	}
	cancelAt, err := parseTimePtrValue(payload.CancelAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cancel_at"})
		return
	}
	periodStart, err := parseTime(payload.CurrentPeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_current_period_start"})
		return
	}
	periodEnd, err := parseTime(payload.CurrentPeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_current_period_end"})
		return
	}

	resp, err := h.subscriptions.CreateSubscription(ctx, subscriptiondomain.CreateSubscriptionRequest{
		CustomerID:         strings.TrimSpace(payload.CustomerID),
		PlanID:             strings.TrimSpace(payload.PlanID),
		Currency:           strings.TrimSpace(payload.Currency),
		StartAt:            startAt,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		TrialEnd:           trialEnd,
		CancelAt:           cancelAt,
		Status:             strings.TrimSpace(payload.Status),
		IdempotencyKey:     strings.TrimSpace(payload.IdempotencyKey),
		Items:              payload.Items,
	})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListSubscriptions handles GET /api/v1/subscriptions
func (h *Handler) ListSubscriptions(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.subscriptions.ListSubscriptions(ctx, subscriptiondomain.ListSubscriptionRequest{
		PageToken:  c.Query("page_token"),
		PageSize:   parseInt32(c.Query("page_size")),
		CustomerID: c.Query("customer_id"),
		Status:     c.Query("status"),
	})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetSubscription handles GET /api/v1/subscriptions/:subscription_id
func (h *Handler) GetSubscription(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	subscriptionID := strings.TrimSpace(c.Param("subscription_id"))
	resp, err := h.subscriptions.GetSubscriptionByID(ctx, subscriptiondomain.GetSubscriptionRequest{ID: subscriptionID})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func writeSubscriptionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subscriptiondomain.ErrInvalidOrganization),
		errors.Is(err, subscriptiondomain.ErrInvalidID),
		errors.Is(err, subscriptiondomain.ErrInvalidCustomer),
		errors.Is(err, subscriptiondomain.ErrInvalidPlan),
		errors.Is(err, subscriptiondomain.ErrInvalidPlanPrice),
		errors.Is(err, subscriptiondomain.ErrInvalidStatus),
		errors.Is(err, subscriptiondomain.ErrInvalidCurrency),
		errors.Is(err, subscriptiondomain.ErrInvalidPeriod),
		errors.Is(err, subscriptiondomain.ErrInvalidQuantity),
		errors.Is(err, subscriptiondomain.ErrMissingItems):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, subscriptiondomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
