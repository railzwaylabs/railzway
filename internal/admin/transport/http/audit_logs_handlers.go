package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

func (h *Handler) ListAuditLogs(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	if h.audit == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "audit_not_configured"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	createdFrom, err := parseTimePtr(c.Query("created_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_from"})
		return
	}
	createdTo, err := parseTimePtr(c.Query("created_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_to"})
		return
	}

	resp, err := h.audit.List(ctx, orgID, auditlog.ListRequest{
		PageToken:    c.Query("page_token"),
		PageSize:     int(parseInt32(c.Query("page_size"))),
		Action:       strings.TrimSpace(c.Query("action")),
		ActorType:    strings.TrimSpace(c.Query("actor_type")),
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		ResourceID:   strings.TrimSpace(c.Query("resource_id")),
		RequestID:    strings.TrimSpace(c.Query("request_id")),
		CreatedFrom:  createdFrom,
		CreatedTo:    createdTo,
	})
	if err != nil {
		if err == auditlog.ErrInvalidCursor {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
