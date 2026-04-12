package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
)

type createProductRequest struct {
	Code           string                        `json:"code" binding:"required"`
	Name           string                        `json:"name" binding:"required"`
	Description    *string                       `json:"description,omitempty"`
	Active         *bool                         `json:"active,omitempty"`
	IdempotencyKey string                        `json:"idempotency_key" binding:"required"`
	FeatureIDs     []string                      `json:"feature_ids,omitempty"`
	Plans          []productdomain.CreateProductPlanInput `json:"plans,omitempty"`
}

func (h *Handler) CreateProduct(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createProductRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	resp, err := h.products.Create(ctx, productdomain.CreateProductRequest{
		Code:           strings.TrimSpace(payload.Code),
		Name:           strings.TrimSpace(payload.Name),
		Description:    payload.Description,
		Active:         payload.Active,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
		FeatureIDs:     payload.FeatureIDs,
		Plans:          payload.Plans,
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type updateProductRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Active      *bool     `json:"active,omitempty"`
	FeatureIDs  *[]string `json:"feature_ids,omitempty"`
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	productID := strings.TrimSpace(c.Param("product_id"))
	var payload updateProductRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	resp, err := h.products.Update(ctx, productID, productdomain.UpdateProductRequest{
		Name:        payload.Name,
		Description: payload.Description,
		Active:      payload.Active,
		FeatureIDs:  payload.FeatureIDs,
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetProduct(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	productID := strings.TrimSpace(c.Param("product_id"))
	expand := strings.TrimSpace(c.Query("expand"))
	resp, err := h.products.GetByID(ctx, productdomain.GetProductRequest{
		ID:             productID,
		ExpandPlans:    strings.Contains(expand, "plans"),
		ExpandFeatures: strings.Contains(expand, "features"),
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListProducts(c *gin.Context) {
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

	expand := strings.TrimSpace(c.Query("expand"))
	resp, err := h.products.List(ctx, productdomain.ListProductRequest{
		PageToken:      c.Query("page_token"),
		PageSize:       parseInt32(c.Query("page_size")),
		Code:           c.Query("code"),
		Name:           c.Query("name"),
		Active:         active,
		ExpandPlans:    strings.Contains(expand, "plans"),
		ExpandFeatures: strings.Contains(expand, "features"),
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
