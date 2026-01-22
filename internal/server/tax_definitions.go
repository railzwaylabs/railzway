package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	taxdomain "github.com/smallbiznis/railzway/internal/tax/domain"
)

type createTaxDefinitionRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	TaxMode     string   `json:"tax_mode"`
	Rate        *float64 `json:"rate"`
	Description *string  `json:"description"`
	IsEnabled   *bool    `json:"is_enabled"`
}

type updateTaxDefinitionRequest struct {
	Name        *string  `json:"name,omitempty"`
	TaxMode     *string  `json:"tax_mode,omitempty"`
	Rate        *float64 `json:"rate,omitempty"`
	Description *string  `json:"description,omitempty"`
}

func (s *Server) CreateTaxDefinition(c *gin.Context) {
	var req createTaxDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.taxSvc.Create(c.Request.Context(), taxdomain.CreateRequest{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		TaxMode:     taxdomain.TaxMode(strings.TrimSpace(req.TaxMode)),
		Rate:        req.Rate,
		Description: trimTaxString(req.Description),
		IsEnabled:   req.IsEnabled,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "tax_definition.create", "tax_definition", &targetID, map[string]any{
			"tax_definition_id": resp.ID,
			"code":              resp.Code,
			"tax_mode":          resp.TaxMode,
			"is_enabled":        resp.IsEnabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) ListTaxDefinitions(c *gin.Context) {
	var query struct {
		Name      string `form:"name"`
		Code      string `form:"code"`
		IsEnabled string `form:"is_enabled"`
		SortBy    string `form:"sort_by"`
		OrderBy   string `form:"order_by"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	isEnabled, err := parseOptionalBool(query.IsEnabled)
	if err != nil {
		AbortWithError(c, newValidationError("is_enabled", "invalid_is_enabled", "invalid is_enabled"))
		return
	}

	resp, err := s.taxSvc.List(c.Request.Context(), taxdomain.ListRequest{
		Name:      strings.TrimSpace(query.Name),
		Code:      strings.TrimSpace(query.Code),
		IsEnabled: isEnabled,
		SortBy:    strings.TrimSpace(query.SortBy),
		OrderBy:   strings.TrimSpace(query.OrderBy),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) UpdateTaxDefinition(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	var req updateTaxDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	var taxMode *taxdomain.TaxMode
	if req.TaxMode != nil {
		trimmed := taxdomain.TaxMode(strings.TrimSpace(*req.TaxMode))
		taxMode = &trimmed
	}

	resp, err := s.taxSvc.Update(c.Request.Context(), taxdomain.UpdateRequest{
		ID:          id,
		Name:        trimTaxString(req.Name),
		TaxMode:     taxMode,
		Rate:        req.Rate,
		Description: trimTaxString(req.Description),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "tax_definition.update", "tax_definition", &targetID, map[string]any{
			"tax_definition_id": resp.ID,
			"code":              resp.Code,
			"tax_mode":          resp.TaxMode,
			"is_enabled":        resp.IsEnabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) DisableTaxDefinition(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.taxSvc.Disable(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "tax_definition.disable", "tax_definition", &targetID, map[string]any{
			"tax_definition_id": resp.ID,
			"code":              resp.Code,
			"is_enabled":        resp.IsEnabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func trimTaxString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
