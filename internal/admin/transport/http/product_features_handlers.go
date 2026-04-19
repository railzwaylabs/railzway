package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
)

type replaceProductFeaturesRequest struct {
	FeatureIDs []string `json:"feature_ids"`
}

func (h *Handler) ListProductFeatures(c *gin.Context) {
	productID := strings.TrimSpace(c.Param("product_id"))
	resp, err := h.productFeatures.List(c.Request.Context(), productfeaturedomain.ListRequest{
		ProductID: productID,
	})
	if err != nil {
		writeProductFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReplaceProductFeatures(c *gin.Context) {
	productID := strings.TrimSpace(c.Param("product_id"))

	var req replaceProductFeaturesRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}

	resp, err := h.productFeatures.Replace(c.Request.Context(), productfeaturedomain.ReplaceRequest{
		ProductID:  productID,
		FeatureIDs: req.FeatureIDs,
	})
	if err != nil {
		writeProductFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
