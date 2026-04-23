package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

// ListInvoices handles GET /api/v1/invoices
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

// GetInvoiceByID handles GET /api/v1/invoices/:invoice_id
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
	case errors.Is(err, invoicedomain.ErrNoBillableItems):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, invoicedomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
