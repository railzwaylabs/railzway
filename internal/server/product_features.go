package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	productfeaturedomain "github.com/smallbiznis/railzway/internal/productfeature/domain"
)

type replaceProductFeaturesRequest struct {
	FeatureIDs []string `json:"feature_ids"`
}

func (s *Server) ListProductFeatures(c *gin.Context) {
	productID := strings.TrimSpace(c.Param("id"))
	resp, err := s.productFeatureSvc.List(c.Request.Context(), productfeaturedomain.ListRequest{
		ProductID: productID,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (s *Server) ReplaceProductFeatures(c *gin.Context) {
	productID := strings.TrimSpace(c.Param("id"))

	var req replaceProductFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.productFeatureSvc.Replace(c.Request.Context(), productfeaturedomain.ReplaceRequest{
		ProductID:  productID,
		FeatureIDs: req.FeatureIDs,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := productID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "product.features.replace", "product", &targetID, map[string]any{
			"product_id":  productID,
			"feature_ids": req.FeatureIDs,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
