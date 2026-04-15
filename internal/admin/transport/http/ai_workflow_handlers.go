package http

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminauth "github.com/railzwaylabs/railzway/internal/admin/auth"
	aiworkflowdomain "github.com/railzwaylabs/railzway/internal/aiworkflow/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type createAIWorkflowRequest struct {
	Title       string                   `json:"title"`
	Summary     string                   `json:"summary"`
	Intent      string                   `json:"intent"`
	SourceRunID string                   `json:"source_run_id"`
	Actions     []createAIWorkflowAction `json:"actions"`
}

type createAIWorkflowAction struct {
	Type    string          `json:"type"`
	Label   string          `json:"label"`
	Payload json.RawMessage `json:"payload"`
}

type approveAIWorkflowRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (h *Handler) CreateAIWorkflow(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	var payload createAIWorkflowRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))
	actions := make([]aiworkflowdomain.CreateWorkflowAction, 0, len(payload.Actions))
	for _, action := range payload.Actions {
		actions = append(actions, aiworkflowdomain.CreateWorkflowAction{
			Type:    aiworkflowdomain.ActionType(action.Type),
			Label:   action.Label,
			Payload: action.Payload,
		})
	}

	resp, err := h.aiWorkflow.CreateWorkflow(ctx, aiworkflowdomain.CreateWorkflowRequest{
		Title:       payload.Title,
		Summary:     payload.Summary,
		Intent:      payload.Intent,
		SourceRunID: payload.SourceRunID,
		Actions:     actions,
	})
	if err != nil {
		writeAIWorkflowError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListAIWorkflows(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	resp, err := h.aiWorkflow.ListWorkflows(ctx, aiworkflowdomain.ListWorkflowsRequest{
		PageToken: c.Query("page_token"),
		PageSize:  parseInt32(c.Query("page_size")),
	})
	if err != nil {
		writeAIWorkflowError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAIWorkflow(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	resp, err := h.aiWorkflow.GetWorkflow(ctx, aiworkflowdomain.GetWorkflowRequest{ID: c.Param("workflow_id")})
	if err != nil {
		writeAIWorkflowError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ApproveAIWorkflow(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	var payload approveAIWorkflowRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))
	resp, err := h.aiWorkflow.ApproveWorkflow(ctx, aiworkflowdomain.ApproveWorkflowRequest{
		ID:      c.Param("workflow_id"),
		ActorID: actorIDFromContext(c),
		Note:    payload.Note,
		Status:  aiworkflowdomain.ApprovalStatus(payload.Status),
	})
	if err != nil {
		writeAIWorkflowError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ExecuteAIWorkflow(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))
	resp, err := h.aiWorkflow.ExecuteWorkflow(ctx, aiworkflowdomain.ExecuteWorkflowRequest{ID: c.Param("workflow_id")})
	if err != nil {
		writeAIWorkflowError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func actorIDFromContext(c *gin.Context) string {
	if val, ok := c.Get(adminauth.CtxUserID); ok {
		switch v := val.(type) {
		case uuid.UUID:
			if v != uuid.Nil {
				return v.String()
			}
		case string:
			if parsed, err := uuid.Parse(v); err == nil && parsed != uuid.Nil {
				return parsed.String()
			}
		}
	}
	return ""
}
