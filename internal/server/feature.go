package server

import (
	"strings"

	"github.com/gin-gonic/gin"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
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

// @Summary      Create Feature
// @Description  Create a new feature
// @Tags         features
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body createFeatureRequest true "Create Feature Request"
// @Success      200  {object}  DataResponse
// @Router       /features [post]
func (s *Server) CreateFeature(c *gin.Context) {
	var req createFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.featureSvc.Create(c.Request.Context(), featuredomain.CreateRequest{
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		Description:    trimFeatureString(req.Description),
		FeatureType:    featuredomain.FeatureType(strings.TrimSpace(req.FeatureType)),
		MeterID:        trimFeatureString(req.MeterID),
		Active:         req.Active,
		Metadata:       req.Metadata,
		IdempotencyKey: idempotencyKeyFromHeader(c),
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

	respondData(c, resp)
}

// @Summary      List Features
// @Description  List available features
// @Tags         features
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        name          query  string  false  "Name"
// @Param        code          query  string  false  "Code"
// @Param        feature_type  query  string  false  "Feature Type"
// @Param        active        query  bool    false  "Active"
// @Param        sort_by       query  string  false  "Sort By"
// @Param        order_by      query  string  false  "Order By"
// @Param        page_token    query  string  false  "Page Token"
// @Param        page_size     query  int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /features [get]
func (s *Server) ListFeatures(c *gin.Context) {
	var query struct {
		pagination.Pagination
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
		PageToken:   query.PageToken,
		PageSize:    int32(query.PageSize),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Features, &resp.PageInfo)
}

// @Summary      Update Feature
// @Description  Update a feature
// @Tags         features
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Feature ID"
// @Param        request  body  updateFeatureRequest  true  "Update Feature Request"
// @Success      200  {object}  DataResponse
// @Router       /features/{id} [patch]
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

	respondData(c, resp)
}

// @Summary      Archive Feature
// @Description  Archive a feature
// @Tags         features
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Feature ID"
// @Success      200  {object}  DataResponse
// @Router       /features/{id}/archive [post]
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

	respondData(c, resp)
}

func trimFeatureString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
