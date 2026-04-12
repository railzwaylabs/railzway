package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
)

type Handler struct {
	subscriptions subscriptiondomain.Service
	invoices      domain.Service
}

func NewHandler(subscriptions subscriptiondomain.Service, invoices domain.Service) *Handler {
	return &Handler{subscriptions: subscriptions, invoices: invoices}
}

func RegisterRoutes(r *gin.Engine, h *Handler) {
	group := r.Group("/customer/v1")
	group.GET("/subscriptions", h.ListSubscriptions)
	group.GET("/subscriptions/:subscription_id", h.GetSubscription)
	group.GET("/invoices", h.ListInvoices)
	group.GET("/invoices/:invoice_id", h.GetInvoice)
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	customerID, ok := customerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_customer_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.subscriptions.ListSubscriptions(ctx, subscriptiondomain.ListSubscriptionRequest{
		PageToken:  c.Query("page_token"),
		PageSize:   parseInt32(c.Query("page_size")),
		CustomerID: customerID.String(),
		Status:     strings.TrimSpace(c.Query("status")),
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
	customerID, ok := customerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_customer_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	subID := strings.TrimSpace(c.Param("subscription_id"))
	resp, err := h.subscriptions.GetSubscriptionByID(ctx, subscriptiondomain.GetSubscriptionRequest{ID: subID})
	if err != nil {
		writeSubscriptionError(c, err)
		return
	}
	if resp.CustomerID != customerID.String() {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListInvoices(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	customerID, ok := customerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_customer_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.invoices.ListInvoices(ctx, domain.ListInvoicesRequest{
		PageToken:  c.Query("page_token"),
		PageSize:   parseInt32(c.Query("page_size")),
		CustomerID: customerID.String(),
		Status:     strings.TrimSpace(c.Query("status")),
		Number:     strings.TrimSpace(c.Query("number")),
	})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	customerID, ok := customerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_customer_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	resp, err := h.invoices.GetInvoice(ctx, domain.GetInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	if resp.Invoice.CustomerID != customerID {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func orgIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	raw := strings.TrimSpace(c.GetHeader("X-Org-ID"))
	if raw == "" {
		raw = strings.TrimSpace(c.Query("org_id"))
	}
	if raw == "" {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

func customerIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	raw := strings.TrimSpace(c.GetHeader("X-Customer-ID"))
	if raw == "" {
		raw = strings.TrimSpace(c.Query("customer_id"))
	}
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}
