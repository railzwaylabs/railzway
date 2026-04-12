package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

type createMeterRequest struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Aggregation    string `json:"aggregation"`
	Unit           string `json:"unit"`
	Active         *bool  `json:"active"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handler) CreateMeter(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createMeterRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	resp, err := h.usage.CreateMeter(ctx, usagedomain.CreateMeterRequest{
		Code:           strings.TrimSpace(payload.Code),
		Name:           strings.TrimSpace(payload.Name),
		Aggregation:    strings.TrimSpace(payload.Aggregation),
		Unit:           strings.TrimSpace(payload.Unit),
		Active:         payload.Active,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type updateMeterRequest struct {
	Name        *string `json:"name"`
	Aggregation *string `json:"aggregation"`
	Unit        *string `json:"unit"`
	Active      *bool   `json:"active"`
}

func (h *Handler) UpdateMeter(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	meterID := strings.TrimSpace(c.Param("meter_id"))
	var payload updateMeterRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	resp, err := h.usage.UpdateMeter(ctx, meterID, usagedomain.UpdateMeterRequest{
		Name:        payload.Name,
		Aggregation: payload.Aggregation,
		Unit:        payload.Unit,
		Active:      payload.Active,
	})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetMeter(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	meterID := strings.TrimSpace(c.Param("meter_id"))
	resp, err := h.usage.GetMeterByID(ctx, usagedomain.GetMeterRequest{ID: meterID})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMeters(c *gin.Context) {
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

	resp, err := h.usage.ListMeters(ctx, usagedomain.ListMeterRequest{
		PageToken: c.Query("page_token"),
		PageSize:  parseInt32(c.Query("page_size")),
		Code:      c.Query("code"),
		Name:      c.Query("name"),
		Active:    active,
	})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
