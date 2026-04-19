package http

import (
	"errors"
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

// CreateCustomer handles POST /api/v1/customers
func (h *Handler) CreateCustomer(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

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

// ListCustomers handles GET /api/v1/customers
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

// GetCustomer handles GET /api/v1/customers/:customer_id
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
