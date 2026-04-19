package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type createCustomerRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	ExternalID     string `json:"external_id"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createCustomerRequest
	if !bindJSONOrAbort(c, &payload) {
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

type updateCustomerRequest struct {
	Name       *string `json:"name"`
	Email      *string `json:"email"`
	ExternalID *string `json:"external_id"`
	Currency   *string `json:"currency"`
}

func (h *Handler) UpdateCustomer(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	customerID := strings.TrimSpace(c.Param("customer_id"))
	var payload updateCustomerRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.customers.Update(ctx, customerID, customerdomain.UpdateCustomerRequest{
		Name:       payload.Name,
		Email:      payload.Email,
		ExternalID: payload.ExternalID,
		Currency:   payload.Currency,
	})
	if err != nil {
		writeCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetCustomer(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
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

func (h *Handler) ListCustomers(c *gin.Context) {
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
