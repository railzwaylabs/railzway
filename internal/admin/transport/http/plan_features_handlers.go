package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	planfeaturedomain "github.com/railzwaylabs/railzway/internal/planfeature/domain"
)

type replacePlanFeaturesRequest struct {
	Features []planfeaturedomain.ReplaceFeatureInput `json:"features"`
}

func (h *Handler) ListPlanFeatures(c *gin.Context) {
	planID := strings.TrimSpace(c.Param("plan_id"))
	resp, err := h.planFeatures.List(c.Request.Context(), planfeaturedomain.ListRequest{
		PlanID: planID,
	})
	if err != nil {
		writePlanFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReplacePlanFeatures(c *gin.Context) {
	planID := strings.TrimSpace(c.Param("plan_id"))

	var req replacePlanFeaturesRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}

	resp, err := h.planFeatures.Replace(c.Request.Context(), planfeaturedomain.ReplaceRequest{
		PlanID:   planID,
		Features: req.Features,
	})
	if err != nil {
		writePlanFeatureError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
