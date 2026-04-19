package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
)

type createSubscriptionRequest struct {
	CustomerID         string                        `json:"customer_id"`
	PlanID             string                        `json:"plan_id"`
	Currency           string                        `json:"currency"`
	StartAt            string                        `json:"start_at"`
	CurrentPeriodStart string                        `json:"current_period_start"`
	CurrentPeriodEnd   string                        `json:"current_period_end"`
	TrialEnd           string                        `json:"trial_end"`
	CancelAt           string                        `json:"cancel_at"`
	Status             string                        `json:"status"`
	IdempotencyKey     string                        `json:"idempotency_key"`
	Items              []createSubscriptionItemInput `json:"items"`
}

type createSubscriptionItemInput struct {
	PlanPriceID    string  `json:"plan_price_id"`
	Quantity       float64 `json:"quantity"`
	StartAt        string  `json:"start_at"`
	EndAt          string  `json:"end_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createSubscriptionRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	currentStart, err := parseTime(payload.CurrentPeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_current_period_start"})
		return
	}
	currentEnd, err := parseTime(payload.CurrentPeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_current_period_end"})
		return
	}
	startAt, err := parseTimePtr(payload.StartAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_start_at"})
		return
	}
	trialEnd, err := parseTimePtr(payload.TrialEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_trial_end"})
		return
	}
	cancelAt, err := parseTimePtr(payload.CancelAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cancel_at"})
		return
	}

	items := make([]subscriptiondomain.CreateSubscriptionItemInput, 0, len(payload.Items))
	for _, item := range payload.Items {
		startAtItem, err := parseTimePtr(item.StartAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_item_start_at"})
			return
		}
		endAtItem, err := parseTimePtr(item.EndAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_item_end_at"})
			return
		}
		items = append(items, subscriptiondomain.CreateSubscriptionItemInput{
			PlanPriceID:    strings.TrimSpace(item.PlanPriceID),
			Quantity:       item.Quantity,
			StartAt:        startAtItem,
			EndAt:          endAtItem,
			IdempotencyKey: strings.TrimSpace(item.IdempotencyKey),
		})
	}

	resp, err := h.subscriptions.CreateSubscription(ctx, subscriptiondomain.CreateSubscriptionRequest{
		CustomerID:         strings.TrimSpace(payload.CustomerID),
		PlanID:             strings.TrimSpace(payload.PlanID),
		Currency:           strings.TrimSpace(payload.Currency),
		StartAt:            startAt,
		CurrentPeriodStart: currentStart,
		CurrentPeriodEnd:   currentEnd,
		TrialEnd:           trialEnd,
		CancelAt:           cancelAt,
		Status:             strings.TrimSpace(payload.Status),
		IdempotencyKey:     strings.TrimSpace(payload.IdempotencyKey),
		Items:              items,
	})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type createSubscriptionItemRequest struct {
	PlanPriceID    string  `json:"plan_price_id"`
	Quantity       float64 `json:"quantity"`
	StartAt        string  `json:"start_at"`
	EndAt          string  `json:"end_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

func (h *Handler) CreateSubscriptionItem(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	subscriptionID := strings.TrimSpace(c.Param("subscription_id"))
	var payload createSubscriptionItemRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	startAt, err := parseTimePtr(payload.StartAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_start_at"})
		return
	}
	endAt, err := parseTimePtr(payload.EndAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_end_at"})
		return
	}

	resp, err := h.subscriptions.CreateSubscriptionItem(ctx, subscriptiondomain.CreateSubscriptionItemRequest{
		SubscriptionID: subscriptionID,
		PlanPriceID:    strings.TrimSpace(payload.PlanPriceID),
		Quantity:       payload.Quantity,
		StartAt:        startAt,
		EndAt:          endAt,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type updateSubscriptionRequest struct {
	Status     *string `json:"status"`
	CancelAt   string  `json:"cancel_at"`
	CanceledAt string  `json:"canceled_at"`
	EndedAt    string  `json:"ended_at"`
}

func (h *Handler) UpdateSubscription(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	subscriptionID := strings.TrimSpace(c.Param("subscription_id"))
	var payload updateSubscriptionRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	cancelAt, err := parseTimePtr(payload.CancelAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_cancel_at"})
		return
	}
	canceledAt, err := parseTimePtr(payload.CanceledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_canceled_at"})
		return
	}
	endedAt, err := parseTimePtr(payload.EndedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_ended_at"})
		return
	}

	resp, err := h.subscriptions.UpdateSubscription(ctx, subscriptionID, subscriptiondomain.UpdateSubscriptionRequest{
		Status:     payload.Status,
		CancelAt:   cancelAt,
		CanceledAt: canceledAt,
		EndedAt:    endedAt,
	})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetSubscription(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
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

func (h *Handler) ListSubscriptions(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
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

func (h *Handler) GetSubscriptionItem(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	itemID := strings.TrimSpace(c.Param("item_id"))
	resp, err := h.subscriptions.GetSubscriptionItemByID(ctx, subscriptiondomain.GetSubscriptionItemRequest{ID: itemID})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListSubscriptionItems(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.subscriptions.ListSubscriptionItems(ctx, subscriptiondomain.ListSubscriptionItemRequest{
		PageToken:      c.Query("page_token"),
		PageSize:       parseInt32(c.Query("page_size")),
		SubscriptionID: strings.TrimSpace(c.Param("subscription_id")),
		PlanPriceID:    c.Query("plan_price_id"),
	})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
