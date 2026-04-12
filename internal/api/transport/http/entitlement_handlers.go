package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	entitlementdomain "github.com/railzwaylabs/railzway/internal/entitlement/domain"
)

// CheckEntitlement handles GET /api/v1/customers/:customer_id/entitlements/:feature_code
func (h *Handler) CheckEntitlement(c *gin.Context) {
	if h.entitlement == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "entitlement_not_configured"})
		return
	}

	customerID := c.Param("customer_id")
	featureCode := c.Param("feature_code")

	resp, err := h.entitlement.CheckEntitlement(c.Request.Context(), entitlementdomain.CheckEntitlementRequest{
		CustomerID:  customerID,
		FeatureCode: featureCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, entitlementdomain.ErrInvalidOrganization):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_organization"})
		case errors.Is(err, entitlementdomain.ErrInvalidCustomer):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_customer"})
		case errors.Is(err, entitlementdomain.ErrInvalidFeature):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_feature"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "detail": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}
