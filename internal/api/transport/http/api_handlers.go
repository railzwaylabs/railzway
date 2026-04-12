package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

// ── Request types ─────────────────────────────────────────────────────────────

type ingestUsageRequest struct {
	MeterCode      string  `json:"meter_code"`
	CustomerID     string  `json:"customer_id"`
	Value          float64 `json:"value"`
	RecordedAt     string  `json:"recorded_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

type createCustomerRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	ExternalID     string `json:"external_id"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

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

// ── Usage events ──────────────────────────────────────────────────────────────

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
	resp, err := h.usage.IngestUsage(ctx, usagedomain.IngestUsageRequest{
		MeterCode:      strings.TrimSpace(payload.MeterCode),
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

// ── Customers ─────────────────────────────────────────────────────────────────

func (h *Handler) CreateCustomer(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	var payload createCustomerRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	resp, err := h.customers.Create(ctx, customerdomain.CreateCustomerRequest{
		Name:           strings.TrimSpace(payload.Name),
		Email:          strings.TrimSpace(payload.Email),
		ExternalID:     strings.TrimSpace(payload.ExternalID),
		Currency:       strings.TrimSpace(payload.Currency),
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListCustomers(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	createdFrom, err := parseTimePtr(c.Query("created_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_from"})
		return
	}
	createdTo, err := parseTimePtr(c.Query("created_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_to"})
		return
	}
	resp, err := h.customers.List(ctx, customerdomain.ListCustomerRequest{
		PageToken:   c.Query("page_token"),
		PageSize:    parseInt32(c.Query("page_size")),
		Name:        c.Query("name"),
		Email:       c.Query("email"),
		Currency:    c.Query("currency"),
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetCustomer(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	customerID := strings.TrimSpace(c.Param("customer_id"))
	resp, err := h.customers.GetByID(ctx, customerdomain.GetCustomerRequest{ID: customerID})
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ── Subscriptions ─────────────────────────────────────────────────────────────

func (h *Handler) CreateSubscription(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	var payload createSubscriptionRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
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

// ── Invoices ──────────────────────────────────────────────────────────────────

func (h *Handler) ListInvoices(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
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
	issuedFrom, err := parseTimePtr(c.Query("issued_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_issued_from"})
		return
	}
	issuedTo, err := parseTimePtr(c.Query("issued_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_issued_to"})
		return
	}
	createdFrom, err := parseTimePtr(c.Query("created_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_from"})
		return
	}
	createdTo, err := parseTimePtr(c.Query("created_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_to"})
		return
	}

	resp, err := h.invoices.ListInvoices(ctx, invoicedomain.ListInvoicesRequest{
		PageToken:       c.Query("page_token"),
		PageSize:        parseInt32(c.Query("page_size")),
		CustomerID:      c.Query("customer_id"),
		SubscriptionID:  c.Query("subscription_id"),
		Status:          c.Query("status"),
		Number:          c.Query("number"),
		PeriodStartFrom: periodStartFrom,
		PeriodStartTo:   periodStartTo,
		IssuedFrom:      issuedFrom,
		IssuedTo:        issuedTo,
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetInvoiceByID(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	resp, err := h.invoices.GetInvoice(ctx, invoicedomain.GetInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func parseInt32(value string) int32 {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return int32(parsed)
}

func parseTime(value string) (time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, errors.New("missing_time")
	}
	return time.Parse(time.RFC3339, raw)
}

func parseTimePtr(value string) (*time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseTimePtrValue(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	return parseTimePtr(*value)
}

// ── Error writers ─────────────────────────────────────────────────────────────

func writeCustomerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, customerdomain.ErrInvalidOrganization),
		errors.Is(err, customerdomain.ErrInvalidID),
		errors.Is(err, customerdomain.ErrInvalidName),
		errors.Is(err, customerdomain.ErrInvalidEmail),
		errors.Is(err, customerdomain.ErrInvalidCurrency):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, customerdomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
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

func writeInvoiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, invoicedomain.ErrInvalidOrganization),
		errors.Is(err, invoicedomain.ErrInvalidSubscription),
		errors.Is(err, invoicedomain.ErrInvalidCustomer),
		errors.Is(err, invoicedomain.ErrInvalidPeriod),
		errors.Is(err, invoicedomain.ErrInvalidStatus),
		errors.Is(err, invoicedomain.ErrInvalidCursor):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, invoicedomain.ErrUsageNotReady):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, invoicedomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
