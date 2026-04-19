package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/authz"
)

type authzPolicyRequest struct {
	Subject string `json:"subject" binding:"required"`
	Object  string `json:"object" binding:"required"`
	Action  string `json:"action" binding:"required"`
}

func (h *Handler) ListAuthzPolicies(c *gin.Context) {
	if h.adminAuthz == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "authz_not_configured"})
		return
	}
	policies, err := h.adminAuthz.ListPolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (h *Handler) AddAuthzPolicy(c *gin.Context) {
	if h.adminAuthz == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "authz_not_configured"})
		return
	}
	var req authzPolicyRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}
	policy := authz.Policy{
		Subject: strings.TrimSpace(req.Subject),
		Object:  strings.TrimSpace(req.Object),
		Action:  strings.TrimSpace(req.Action),
	}
	added, err := h.adminAuthz.AddPolicy(policy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"added": added})
}

func (h *Handler) RemoveAuthzPolicy(c *gin.Context) {
	if h.adminAuthz == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "authz_not_configured"})
		return
	}
	var req authzPolicyRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}
	policy := authz.Policy{
		Subject: strings.TrimSpace(req.Subject),
		Object:  strings.TrimSpace(req.Object),
		Action:  strings.TrimSpace(req.Action),
	}
	removed, err := h.adminAuthz.RemovePolicy(policy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": removed})
}
