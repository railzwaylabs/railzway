package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

type createSubscriptionItemRequest struct {
	PriceID  string  `json:"price_id"`
	MeterID  *string `json:"meter_id,omitempty"`
	Quantity int8    `json:"quantity,omitempty"`
}

type createSubscriptionRequest struct {
	CustomerID       string                                        `json:"customer_id"`
	CollectionMode   subscriptiondomain.SubscriptionCollectionMode `json:"collection_mode"`
	BillingCycleType string                                        `json:"billing_cycle_type"`
	Items            []createSubscriptionItemRequest               `json:"items"`
	TrialDays        *int                                          `json:"trial_days,omitempty"`
	Metadata         map[string]any                                `json:"metadata,omitempty"`
}

// @Summary      Create Subscription
// @Description  Create a new subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body subscriptiondomain.CreateSubscriptionRequest true "Create Subscription Request"
// @Success      200  {object}  DataResponse
// @Router       /subscriptions [post]
func (s *Server) CreateSubscription(c *gin.Context) {
	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	if err := rejectSubscriptionMeterID(req.Items); err != nil {
		AbortWithError(c, err)
		return
	}

	resp, err := s.subscriptionSvc.Create(c.Request.Context(), subscriptiondomain.CreateSubscriptionRequest{
		CustomerID:       strings.TrimSpace(req.CustomerID),
		CollectionMode:   req.CollectionMode,
		BillingCycleType: strings.TrimSpace(req.BillingCycleType),
		Items:            normalizeSubscriptionItems(req.Items),
		Metadata:         req.Metadata,
		IdempotencyKey:   idempotencyKeyFromHeader(c),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "subscription.create", "subscription", &targetID, map[string]any{
			"subscription_id": resp.ID,
			"customer_id":     resp.CustomerID,
			"status":          string(resp.Status),
		})
	}

	respondData(c, resp)
}

type replaceSubscriptionItemsRequest struct {
	Items []createSubscriptionItemRequest `json:"items"`
}

// @Summary      Replace Subscription Items
// @Description  Replace items in a subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id       path      string                           true  "Subscription ID"
// @Param        request  body      replaceSubscriptionItemsRequest  true  "Replace Subscription Items Request"
// @Success      200  {object}  DataResponse
// @Router       /subscriptions/{id}/items [put]
func (s *Server) ReplaceSubscriptionItems(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if _, err := snowflake.ParseString(id); err != nil {
		AbortWithError(c, newValidationError("id", "invalid_id", "invalid id"))
		return
	}

	var req replaceSubscriptionItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	if err := rejectSubscriptionMeterID(req.Items); err != nil {
		AbortWithError(c, err)
		return
	}

	resp, err := s.subscriptionSvc.ReplaceItems(c.Request.Context(), subscriptiondomain.ReplaceSubscriptionItemsRequest{
		SubscriptionID: id,
		Items:          normalizeSubscriptionItems(req.Items),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "subscription.items.replace", "subscription", &targetID, map[string]any{
			"subscription_id": resp.ID,
		})
	}

	respondData(c, resp)
}

// @Summary      List Subscriptions
// @Description  List available subscriptions
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        status        query     string  false  "Status"
// @Param        customer_id   query     string  false  "Customer ID"
// @Param        created_from  query     string  false  "Created From"
// @Param        created_to    query     string  false  "Created To"
// @Param        page_token    query     string  false  "Page Token"
// @Param        page_size     query     int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /subscriptions [get]
func (s *Server) ListSubscriptions(c *gin.Context) {
	var query struct {
		pagination.Pagination
		Status      string `form:"status"`
		CustomerID  string `form:"customer_id"`
		CreatedFrom string `form:"created_from"`
		CreatedTo   string `form:"created_to"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	createdFrom, err := parseOptionalTime(query.CreatedFrom, false)
	if err != nil {
		AbortWithError(c, newValidationError("created_from", "invalid_created_from", "invalid created_from"))
		return
	}

	createdTo, err := parseOptionalTime(query.CreatedTo, true)
	if err != nil {
		AbortWithError(c, newValidationError("created_to", "invalid_created_to", "invalid created_to"))
		return
	}

	resp, err := s.subscriptionSvc.List(c.Request.Context(), subscriptiondomain.ListSubscriptionRequest{
		Status:      strings.TrimSpace(query.Status),
		CustomerID:  strings.TrimSpace(query.CustomerID),
		PageToken:   query.PageToken,
		PageSize:    int32(query.PageSize),
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Subscriptions, &resp.PageInfo)
}

// @Summary      Get Subscription
// @Description  Get subscription by ID
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Subscription ID"
// @Success      200  {object}  DataResponse
// @Router       /subscriptions/{id} [get]
func (s *Server) GetSubscriptionByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if _, err := snowflake.ParseString(id); err != nil {
		AbortWithError(c, newValidationError("id", "invalid_id", "invalid id"))
		return
	}

	item, err := s.subscriptionSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, item)
}

// @Summary      List Subscription Entitlements
// @Description  List entitlements for a subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id            path     string  true   "Subscription ID"
// @Param        effective_at  query    string  false  "Effective At (RFC3339 or YYYY-MM-DD)"
// @Param        page_token    query    string  false  "Page Token"
// @Param        page_size     query    int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /subscriptions/{id}/entitlements [get]
func (s *Server) ListSubscriptionEntitlements(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if _, err := snowflake.ParseString(id); err != nil {
		AbortWithError(c, newValidationError("id", "invalid_id", "invalid id"))
		return
	}

	var query struct {
		pagination.Pagination
		EffectiveAt string `form:"effective_at"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	effectiveAt, err := parseOptionalTime(query.EffectiveAt, false)
	if err != nil {
		AbortWithError(c, newValidationError("effective_at", "invalid_effective_at", "invalid effective_at"))
		return
	}

	resp, err := s.subscriptionSvc.ListEntitlements(c.Request.Context(), subscriptiondomain.ListEntitlementsRequest{
		SubscriptionID: id,
		EffectiveAt:    effectiveAt,
		PageToken:      query.PageToken,
		PageSize:       int32(query.PageSize),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Entitlements, &resp.PageInfo)
}

// @Summary      Cancel Subscription
// @Description  Cancel a subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Subscription ID"
// @Success      204
// @Router       /subscriptions/{id}/cancel [post]
func (s *Server) CancelSubscription(c *gin.Context) {
	s.transitionSubscription(
		c,
		subscriptiondomain.SubscriptionStatusCanceled,
		"subscription.cancel",
	)
}

// @Summary      Activate Subscription
// @Description  Activate a subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Subscription ID"
// @Success      204
// @Router       /subscriptions/{id}/activate [post]
func (s *Server) ActivateSubscription(c *gin.Context) {
	s.transitionSubscription(
		c,
		subscriptiondomain.SubscriptionStatusActive,
		"subscription.activate",
	)
}

// @Summary      Pause Subscription
// @Description  Pause a subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Subscription ID"
// @Success      204
// @Router       /subscriptions/{id}/pause [post]
func (s *Server) PauseSubscription(c *gin.Context) {
	s.transitionSubscription(
		c,
		subscriptiondomain.SubscriptionStatusPaused,
		"subscription.pause",
	)
}

// @Summary      Resume Subscription
// @Description  Resume a subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Subscription ID"
// @Success      204
// @Router       /subscriptions/{id}/resume [post]
func (s *Server) ResumeSubscription(c *gin.Context) {
	s.transitionSubscription(
		c,
		subscriptiondomain.SubscriptionStatusActive,
		"subscription.resume",
	)
}

func (s *Server) transitionSubscription(c *gin.Context, target subscriptiondomain.SubscriptionStatus, auditAction string) {
	id := strings.TrimSpace(c.Param("id"))
	if _, err := snowflake.ParseString(id); err != nil {
		AbortWithError(c, newValidationError("id", "invalid_id", "invalid id"))
		return
	}

	if err := s.subscriptionSvc.TransitionSubscription(
		c.Request.Context(),
		id,
		target,
		"",
	); err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil && strings.TrimSpace(auditAction) != "" {
		targetID := id
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, auditAction, "subscription", &targetID, map[string]any{
			"subscription_id": id,
			"status":          string(target),
		})
	}

	c.Status(http.StatusNoContent)
}

func normalizeSubscriptionItems(items []createSubscriptionItemRequest) []subscriptiondomain.CreateSubscriptionItemRequest {
	if len(items) == 0 {
		return nil
	}

	normalized := make([]subscriptiondomain.CreateSubscriptionItemRequest, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, subscriptiondomain.CreateSubscriptionItemRequest{
			PriceID:  strings.TrimSpace(item.PriceID),
			Quantity: item.Quantity,
		})
	}
	return normalized
}

func rejectSubscriptionMeterID(items []createSubscriptionItemRequest) error {
	for _, item := range items {
		if item.MeterID == nil {
			continue
		}
		if strings.TrimSpace(*item.MeterID) == "" {
			continue
		}
		return newValidationError("items.meter_id", "unsupported", "meter_id is not supported on subscription items")
	}
	return nil
}

func isSubscriptionValidationError(err error) bool {
	switch {
	case errors.Is(err, subscriptiondomain.ErrInvalidOrganization),
		errors.Is(err, subscriptiondomain.ErrInvalidCustomer),
		errors.Is(err, subscriptiondomain.ErrInvalidSubscription),
		errors.Is(err, subscriptiondomain.ErrInvalidMeterID),
		errors.Is(err, subscriptiondomain.ErrInvalidMeterCode),
		errors.Is(err, subscriptiondomain.ErrInvalidStatus),
		errors.Is(err, subscriptiondomain.ErrInvalidTargetStatus),
		errors.Is(err, subscriptiondomain.ErrInvalidTransition),
		errors.Is(err, subscriptiondomain.ErrMissingSubscriptionItems),
		errors.Is(err, subscriptiondomain.ErrMissingPricing),
		errors.Is(err, subscriptiondomain.ErrMissingCustomer),
		errors.Is(err, subscriptiondomain.ErrBillingCyclesOpen),
		errors.Is(err, subscriptiondomain.ErrInvoicesNotFinalized),
		errors.Is(err, subscriptiondomain.ErrInvalidCollectionMode),
		errors.Is(err, subscriptiondomain.ErrInvalidBillingCycleType),
		errors.Is(err, subscriptiondomain.ErrInvalidCurrency),
		errors.Is(err, subscriptiondomain.ErrInvalidStartAt),
		errors.Is(err, subscriptiondomain.ErrInvalidPeriod),
		errors.Is(err, subscriptiondomain.ErrInvalidItems),
		errors.Is(err, subscriptiondomain.ErrInvalidQuantity),
		errors.Is(err, subscriptiondomain.ErrInvalidPrice),
		errors.Is(err, subscriptiondomain.ErrInvalidProduct),
		errors.Is(err, subscriptiondomain.ErrMultipleFlatPrices),
		errors.Is(err, subscriptiondomain.ErrMissingEntitlements):
		return true
	default:
		return false
	}
}
