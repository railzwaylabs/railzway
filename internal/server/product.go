package server

import (
	"strings"

	"github.com/gin-gonic/gin"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

type createProductRequest struct {
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Active      *bool          `json:"active"`
	Metadata    map[string]any `json:"metadata"`
}

type updateProductRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Active      *bool          `json:"active,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// @Summary      Create Product
// @Description  Create a new product
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body createProductRequest true "Create Product Request"
// @Success      200  {object}  DataResponse
// @Router       /products [post]
func (s *Server) CreateProduct(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.productSvc.Create(c.Request.Context(), productdomain.CreateRequest{
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		Description:    req.Description,
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
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "product.create", "product", &targetID, map[string]any{
			"product_id": resp.ID,
			"code":       resp.Code,
			"name":       resp.Name,
			"active":     resp.Active,
		})
	}

	respondData(c, resp)
}

// @Summary      List Products
// @Description  List available products
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        name     query     string  false  "Name"
// @Param        active   query     bool    false  "Active"
// @Param        sort_by  query     string  false  "Sort By"
// @Param        order_by query     string  false  "Order By"
// @Param        page_token  query  string  false  "Page Token"
// @Param        page_size   query  int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /products [get]
func (s *Server) ListProducts(c *gin.Context) {
	var query struct {
		pagination.Pagination
		Name    string `form:"name"`
		Active  string `form:"active"`
		SortBy  string `form:"sort_by"`
		OrderBy string `form:"order_by"`
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

	resp, err := s.productSvc.List(c.Request.Context(), productdomain.ListRequest{
		Name:      strings.TrimSpace(query.Name),
		Active:    active,
		SortBy:    strings.TrimSpace(query.SortBy),
		OrderBy:   strings.TrimSpace(query.OrderBy),
		PageToken: query.PageToken,
		PageSize:  int32(query.PageSize),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Products, &resp.PageInfo)
}

// @Summary      Get Product
// @Description  Get product by ID
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Product ID"
// @Success      200  {object}  DataResponse
// @Router       /products/{id} [get]
func (s *Server) GetProductByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.productSvc.Get(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

// @Summary      Update Product
// @Description  Update product details
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id       path      string                true  "Product ID"
// @Param        request  body      updateProductRequest  true  "Update Product Request"
// @Success      200  {object}  DataResponse
// @Router       /products/{id} [patch]
func (s *Server) UpdateProduct(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.productSvc.Update(c.Request.Context(), productdomain.UpdateRequest{
		ID:          id,
		Name:        trimProductString(req.Name),
		Description: trimProductString(req.Description),
		Active:      req.Active,
		Metadata:    req.Metadata,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "product.update", "product", &targetID, map[string]any{
			"product_id": resp.ID,
			"code":       resp.Code,
			"name":       resp.Name,
			"active":     resp.Active,
		})
	}

	respondData(c, resp)
}

// @Summary      Archive Product
// @Description  Archive a product
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Product ID"
// @Success      200  {object}  DataResponse
// @Router       /products/{id}/archive [post]
func (s *Server) ArchiveProduct(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.productSvc.Archive(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "product.archive", "product", &targetID, map[string]any{
			"product_id": resp.ID,
			"code":       resp.Code,
			"active":     resp.Active,
		})
	}

	respondData(c, resp)
}

func isProductValidationError(err error) bool {
	switch err {
	case productdomain.ErrInvalidOrganization,
		productdomain.ErrInvalidCode,
		productdomain.ErrInvalidName,
		productdomain.ErrInvalidID:
		return true
	default:
		return false
	}
}

func trimProductString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
