package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
)

func (h *Handler) ListPayments(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
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

	resp, err := h.payments.ListPayments(ctx, paymentdomain.ListPaymentsRequest{
		PageToken:   c.Query("page_token"),
		PageSize:    parseInt32(c.Query("page_size")),
		CustomerID:  strings.TrimSpace(c.Query("customer_id")),
		InvoiceID:   strings.TrimSpace(c.Query("invoice_id")),
		Status:      strings.TrimSpace(c.Query("status")),
		Provider:    strings.TrimSpace(c.Query("provider")),
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
