package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type createFeatureRequest struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	FeatureType    string  `json:"feature_type"`
	MeterID        *string `json:"meter_id"`
	Active         *bool   `json:"active"`
	IdempotencyKey string  `json:"idempotency_key"`
}

func (h *Handler) CreateFeature(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createFeatureRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.features.Create(ctx, featuredomain.CreateFeatureRequest{
		Code:           strings.TrimSpace(payload.Code),
		Name:           strings.TrimSpace(payload.Name),
		Description:    payload.Description,
		FeatureType:    strings.TrimSpace(payload.FeatureType),
		MeterID:        payload.MeterID,
		Active:         payload.Active,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type updateFeatureRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	FeatureType *string `json:"feature_type"`
	MeterID     *string `json:"meter_id"`
	Active      *bool   `json:"active"`
}

func (h *Handler) UpdateFeature(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	featureID := strings.TrimSpace(c.Param("feature_id"))
	var payload updateFeatureRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.features.Update(ctx, featureID, featuredomain.UpdateFeatureRequest{
		Name:        payload.Name,
		Description: payload.Description,
		FeatureType: payload.FeatureType,
		MeterID:     payload.MeterID,
		Active:      payload.Active,
	})
	if err != nil {
		writeFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetFeature(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	featureID := strings.TrimSpace(c.Param("feature_id"))
	resp, err := h.features.GetByID(ctx, featuredomain.GetFeatureRequest{ID: featureID})
	if err != nil {
		writeFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListFeatures(c *gin.Context) {
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

	resp, err := h.features.List(ctx, featuredomain.ListFeatureRequest{
		PageToken:   c.Query("page_token"),
		PageSize:    parseInt32(c.Query("page_size")),
		Code:        c.Query("code"),
		Name:        c.Query("name"),
		FeatureType: c.Query("feature_type"),
		Active:      active,
	})
	if err != nil {
		writeFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
