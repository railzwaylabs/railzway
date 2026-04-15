package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	aiassistantdomain "github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type createAIAssistantRunRequest struct {
	CustomerRef string `json:"customer_ref"`
	TimeRange   string `json:"time_range"`
	Intent      string `json:"intent"`
	Prompt      string `json:"prompt"`
}

func (h *Handler) AIAssistantOverview(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.aiAssistant.Overview(ctx)
	if err != nil {
		writeAIAssistantError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateAIAssistantRun(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	var payload createAIAssistantRunRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))
	resp, err := h.aiAssistant.CreateRun(ctx, aiassistantdomain.CreateRunRequest{
		CustomerRef: payload.CustomerRef,
		TimeRange:   payload.TimeRange,
		Intent:      aiassistantdomain.Intent(payload.Intent),
		Prompt:      payload.Prompt,
	})
	if err != nil {
		writeAIAssistantError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListAIAssistantRuns(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	resp, err := h.aiAssistant.ListRuns(ctx, aiassistantdomain.ListRunsRequest{
		PageToken: c.Query("page_token"),
		PageSize:  parseInt32(c.Query("page_size")),
	})
	if err != nil {
		writeAIAssistantError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAIAssistantRun(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	resp, err := h.aiAssistant.GetRun(ctx, aiassistantdomain.GetRunRequest{ID: c.Param("run_id")})
	if err != nil {
		writeAIAssistantError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

