package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	featuredomain "github.com/smallbiznis/railzway/internal/feature/domain"
)

type createFeatureRequest struct {
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	FeatureType string         `json:"feature_type"`
	MeterID     *string        `json:"meter_id"`
	Active      *bool          `json:"active"`
	Metadata    map[string]any `json:"metadata"`
}

type updateFeatureRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	FeatureType *string        `json:"feature_type,omitempty"`
	MeterID     *string        `json:"meter_id,omitempty"`
	Active      *bool          `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (s *Server) CreateFeature(c *gin.Context) {
	var req createFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.featureSvc.Create(c.Request.Context(), featuredomain.CreateRequest{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		Description: trimFeatureString(req.Description),
		FeatureType: featuredomain.FeatureType(strings.TrimSpace(req.FeatureType)),
		MeterID:     trimFeatureString(req.MeterID),
		Active:      req.Active,
		Metadata:    req.Metadata,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "feature.create", "feature", &targetID, map[string]any{
			"feature_id":   resp.ID,
			"code":         resp.Code,
			"feature_type": resp.FeatureType,
			"active":       resp.Active,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) ListFeatures(c *gin.Context) {
	var query struct {
		Name        string `form:"name"`
		Code        string `form:"code"`
		FeatureType string `form:"feature_type"`
		Active      string `form:"active"`
		SortBy      string `form:"sort_by"`
		OrderBy     string `form:"order_by"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	active, err := parseOptionalBool(query.Active)
	if err != nil {
		AbortWithError(c, newValidationError("active", "invalid_active", "invalid active"))
		return
	}

	var featureType *featuredomain.FeatureType
	rawType := strings.TrimSpace(query.FeatureType)
	if rawType != "" {
		parsed := featuredomain.FeatureType(strings.ToLower(rawType))
		if parsed != featuredomain.FeatureTypeBoolean && parsed != featuredomain.FeatureTypeMetered {
			AbortWithError(c, newValidationError("feature_type", "invalid_feature_type", "invalid feature type"))
			return
		}
		featureType = &parsed
	}

	resp, err := s.featureSvc.List(c.Request.Context(), featuredomain.ListRequest{
		Name:        strings.TrimSpace(query.Name),
		Code:        strings.TrimSpace(query.Code),
		FeatureType: featureType,
		Active:      active,
		SortBy:      strings.TrimSpace(query.SortBy),
		OrderBy:     strings.TrimSpace(query.OrderBy),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) UpdateFeature(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	var req updateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	var featureType *featuredomain.FeatureType
	if req.FeatureType != nil {
		parsed := featuredomain.FeatureType(strings.TrimSpace(*req.FeatureType))
		featureType = &parsed
	}

	resp, err := s.featureSvc.Update(c.Request.Context(), featuredomain.UpdateRequest{
		ID:          id,
		Name:        trimFeatureString(req.Name),
		Description: trimFeatureString(req.Description),
		FeatureType: featureType,
		MeterID:     trimFeatureString(req.MeterID),
		Active:      req.Active,
		Metadata:    req.Metadata,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "feature.update", "feature", &targetID, map[string]any{
			"feature_id":   resp.ID,
			"code":         resp.Code,
			"feature_type": resp.FeatureType,
			"active":       resp.Active,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) ArchiveFeature(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.featureSvc.Archive(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "feature.archive", "feature", &targetID, map[string]any{
			"feature_id": resp.ID,
			"code":       resp.Code,
			"active":     resp.Active,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func trimFeatureString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
