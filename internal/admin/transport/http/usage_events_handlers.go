package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

func (h *Handler) ListUsageEvents(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
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

type ingestUsageRequest struct {
	MeterCode      string  `json:"meter_code"`
	CustomerID     string  `json:"customer_id"`
	Value          float64 `json:"value"`
	RecordedAt     string  `json:"recorded_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

func (h *Handler) IngestUsage(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload ingestUsageRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	recordedAt, err := parseTime(payload.RecordedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_recorded_at"})
		return
	}

	resp, err := h.usage.IngestUsage(ctx, usagedomain.IngestUsageRequest{
		MeterCode:      payload.MeterCode,
		CustomerID:     payload.CustomerID,
		Value:          payload.Value,
		RecordedAt:     recordedAt,
		IdempotencyKey: payload.IdempotencyKey,
	})
	if err != nil {
		writeUsageError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
