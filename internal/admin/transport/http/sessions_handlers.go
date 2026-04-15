package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminauth "github.com/railzwaylabs/railzway/internal/admin/auth"
)

type sessionListResponse struct {
	Sessions []adminauth.SessionView `json:"sessions"`
}

type revokeSessionResponse struct {
	Status         string `json:"status"`
	RevokedCount   int64  `json:"revokedCount,omitempty"`
	RevokedCurrent bool   `json:"revokedCurrent,omitempty"`
}

func (h *Handler) ListProfileSessions(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessions, err := h.auth.ListUserSessions(c.Request.Context(), userID, h.sessionTokenFromRequest(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, sessionListResponse{Sessions: sessions})
}

func (h *Handler) RevokeProfileSession(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	sessionID, err := uuid.Parse(strings.TrimSpace(c.Param("session_id")))
	if err != nil || sessionID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session"})
		return
	}

	token := h.sessionTokenFromRequest(c)
	currentSessionID, err := h.auth.CurrentSessionID(c.Request.Context(), token)
	if err != nil && err != adminauth.ErrSessionNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	ctx := h.withAuditContext(c, c.Request.Context())
	if err := h.auth.RevokeUserSessionByID(ctx, userID, sessionID); err != nil {
		if err == adminauth.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	revokedCurrent := currentSessionID != uuid.Nil && currentSessionID == sessionID
	if revokedCurrent {
		clearSessionCookie(c, h.cfg)
		clearCSRFCookie(c, h.cfg)
	}

	c.JSON(http.StatusOK, revokeSessionResponse{
		Status:         "ok",
		RevokedCurrent: revokedCurrent,
	})
}

func (h *Handler) RevokeOtherProfileSessions(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := h.withAuditContext(c, c.Request.Context())
	revokedCount, err := h.auth.RevokeOtherUserSessions(ctx, userID, h.sessionTokenFromRequest(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, revokeSessionResponse{
		Status:       "ok",
		RevokedCount: revokedCount,
	})
}

func (h *Handler) ListOrganizationSessions(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	if !h.requireSessionAdminRole(c) {
		return
	}
	orgID, ok := requireOrgParam(c, "org_id")
	if !ok {
		return
	}

	sessions, err := h.auth.ListOrganizationSessions(c.Request.Context(), orgID, h.sessionTokenFromRequest(c))
	if err != nil {
		switch err {
		case adminauth.ErrInvalidOrganization:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		}
		return
	}

	c.JSON(http.StatusOK, sessionListResponse{Sessions: sessions})
}

func (h *Handler) RevokeOrganizationSession(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	if !h.requireSessionAdminRole(c) {
		return
	}
	orgID, ok := requireOrgParam(c, "org_id")
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(strings.TrimSpace(c.Param("session_id")))
	if err != nil || sessionID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session"})
		return
	}

	token := h.sessionTokenFromRequest(c)
	currentSessionID, err := h.auth.CurrentSessionID(c.Request.Context(), token)
	if err != nil && err != adminauth.ErrSessionNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	ctx := h.withAuditContext(c, c.Request.Context())
	if err := h.auth.RevokeOrganizationSession(ctx, orgID, sessionID); err != nil {
		switch err {
		case adminauth.ErrInvalidOrganization:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		case adminauth.ErrSessionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		}
		return
	}

	revokedCurrent := currentSessionID != uuid.Nil && currentSessionID == sessionID
	if revokedCurrent {
		clearSessionCookie(c, h.cfg)
		clearCSRFCookie(c, h.cfg)
	}

	c.JSON(http.StatusOK, revokeSessionResponse{
		Status:         "ok",
		RevokedCurrent: revokedCurrent,
	})
}

func (h *Handler) sessionTokenFromRequest(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	cookieName := ""
	if h.cfg != nil {
		cookieName = strings.TrimSpace(h.cfg.SessionConfig.SessionCookie)
	}
	return extractBearerToken(c.Request, cookieName)
}

func (h *Handler) requireSessionAdminRole(c *gin.Context) bool {
	role, ok := roleFromContext(c)
	if !ok || strings.TrimSpace(role) == "" {
		userID, userOK := userIDFromContext(c)
		orgID, orgOK := orgIDFromContext(c)
		if !userOK || !orgOK || h.auth == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return false
		}
		var err error
		role, err = h.auth.GetMemberRole(c.Request.Context(), userID, orgID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return false
		}
	}

	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "OWNER", "ADMIN":
		return true
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return false
	}
}
