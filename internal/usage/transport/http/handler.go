package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/usage/domain"
)

type Handler struct {
	svc domain.Service
}

func NewHandler(svc domain.Service) *Handler {
	return &Handler{svc: svc}
}

func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.POST("/usage", h.Ingest)
	r.GET("/usage", h.List)
}

type ingestUsageRequest struct {
	MeterCode      string  `json:"meter_code"`
	CustomerID     string  `json:"customer_id"`
	Value          float64 `json:"value"`
	RecordedAt     string  `json:"recorded_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

func (h *Handler) Ingest(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	var payload ingestUsageRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	recordedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.RecordedAt))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_recorded_at"})
		return
	}

	resp, err := h.svc.IngestUsage(ctx, domain.IngestUsageRequest{
		MeterCode:      strings.TrimSpace(payload.MeterCode),
		CustomerID:     strings.TrimSpace(payload.CustomerID),
		Value:          payload.Value,
		RecordedAt:     recordedAt,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) List(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	pageSize := parseInt32(c.Query("page_size"))
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

	resp, err := h.svc.ListUsage(ctx, domain.ListUsageRequest{
		PageToken:    c.Query("page_token"),
		PageSize:     pageSize,
		MeterID:      c.Query("meter_id"),
		CustomerID:   c.Query("customer_id"),
		Status:       c.Query("status"),
		RecordedFrom: recordedFrom,
		RecordedTo:   recordedTo,
	})
	if err != nil {
		writeDomainError(c, err)
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

func writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidOrganization),
		errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidCode),
		errors.Is(err, domain.ErrInvalidName),
		errors.Is(err, domain.ErrInvalidAggregation),
		errors.Is(err, domain.ErrInvalidUnit),
		errors.Is(err, domain.ErrInvalidMeter),
		errors.Is(err, domain.ErrInvalidCustomer),
		errors.Is(err, domain.ErrInvalidValue),
		errors.Is(err, domain.ErrInvalidRecordedAt),
		errors.Is(err, domain.ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
