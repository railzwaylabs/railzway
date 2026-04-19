package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	apikeydomain "github.com/railzwaylabs/railzway/internal/apikey/domain"
)

type createAPIKeyRequest struct {
	Name           string   `json:"name" binding:"required"`
	KeyType        string   `json:"key_type" binding:"required"`
	Scopes         []string `json:"scopes,omitempty"`
	AllowedIPs     []string `json:"allowed_ips,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
}

func (h *Handler) ListAPIKeys(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	resp, err := h.apiKeys.ListKeys(c.Request.Context(), orgID.String())
	if err != nil {
		writeAPIKeysError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateAPIKey(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	var req apikeydomain.CreateAPIKeyRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}
	resp, err := h.apiKeys.CreateKey(c.Request.Context(), orgID.String(), apikeydomain.CreateAPIKeyRequest{
		Name:           strings.TrimSpace(req.Name),
		KeyType:        strings.TrimSpace(req.KeyType),
		Scopes:         req.Scopes,
		AllowedIPs:     req.AllowedIPs,
		AllowedDomains: req.AllowedDomains,
	})
	if err != nil {
		writeAPIKeysError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) RevokeAPIKey(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	resp, err := h.apiKeys.RevokeKey(c.Request.Context(), orgID.String(), id)
	if err != nil {
		writeAPIKeysError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func writeAPIKeysError(c *gin.Context, err error) {
	switch {
	case err == apikeydomain.ErrInvalidOrganization,
		err == apikeydomain.ErrInvalidID,
		err == apikeydomain.ErrInvalidKeyType,
		err == apikeydomain.ErrInvalidName:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err == apikeydomain.ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
