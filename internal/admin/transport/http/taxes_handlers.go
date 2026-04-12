package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	taxdomain "github.com/railzwaylabs/railzway/internal/tax/domain"
)

type createTaxRateRequest struct {
	Code       string          `json:"code"`
	Name       string          `json:"name"`
	Percentage float64         `json:"percentage"`
	Inclusive  *bool           `json:"inclusive"`
	Active     *bool           `json:"active"`
	Metadata   json.RawMessage `json:"metadata"`
}

func (h *Handler) CreateTaxRate(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createTaxRateRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	inclusive := false
	if payload.Inclusive != nil {
		inclusive = *payload.Inclusive
	}
	active := true
	if payload.Active != nil {
		active = *payload.Active
	}

	resp, err := h.taxes.CreateTaxRate(ctx, taxdomain.CreateTaxRateRequest{
		Code:       strings.TrimSpace(payload.Code),
		Name:       strings.TrimSpace(payload.Name),
		Percentage: payload.Percentage,
		Inclusive:  inclusive,
		Active:     active,
		Metadata:   payload.Metadata,
	})
	if err != nil {
		writeTaxError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListTaxes(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	active, err := parseBoolPtr(c.Query("active"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_active"})
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

	resp, err := h.taxes.ListTaxRates(ctx, taxdomain.ListTaxRatesRequest{
		PageToken:   c.Query("page_token"),
		PageSize:    parseInt32(c.Query("page_size")),
		Code:        strings.TrimSpace(c.Query("code")),
		Name:        strings.TrimSpace(c.Query("name")),
		Active:      active,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		writeTaxError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
