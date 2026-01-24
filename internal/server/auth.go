package server

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	auditdomain "github.com/railzwaylabs/railzway/internal/audit/domain"
	authconfig "github.com/railzwaylabs/railzway/internal/auth/config"
	authdomain "github.com/railzwaylabs/railzway/internal/auth/domain"
	authfeatures "github.com/railzwaylabs/railzway/internal/auth/features"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) Login(c *gin.Context) {
	if s.cfg.IsCloud() {
		AbortWithError(c, ErrNotFound)
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	email := strings.TrimSpace(req.Email)
	result, err := s.authsvc.Login(c.Request.Context(), authdomain.LoginRequest{
		Email:     email,
		Password:  req.Password,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		if s.auditSvc != nil {
			_ = s.auditSvc.AuditLog(c.Request.Context(), nil, string(auditdomain.ActorTypeUser), nil, "user.login_failed", "user", nil, map[string]any{
				"email": email,
			})
		}
		AbortWithError(c, err)
		return
	}

	s.sessions.Set(c, result.RawToken, result.ExpiresAt)

	s.enrichSessionMetadata(c, result)

	if s.auditSvc != nil {
		var userID *string
		if result.Session != nil {
			if rawUserID, ok := result.Session.Metadata["user_id"].(string); ok && strings.TrimSpace(rawUserID) != "" {
				trimmed := strings.TrimSpace(rawUserID)
				userID = &trimmed
			}
		}
		targetID := userID
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, string(auditdomain.ActorTypeUser), userID, "user.login", "user", targetID, map[string]any{
			"email": email,
		})
	}

	c.JSON(http.StatusOK, result.Session)
}

type AuthProviderInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	LoginPath   string `json:"login_path"`
}

func (s *Server) AuthProviders(c *gin.Context) {
	cfgs := authconfig.ParseAuthProvidersFromEnv()
	providers := make([]AuthProviderInfo, 0, len(cfgs))
	for _, cfg := range cfgs {
		name := strings.ToLower(strings.TrimSpace(cfg.Type))
		if name == "" || name == "local" {
			continue
		}

		if !cfg.Enabled || !authfeatures.ImplementedAuthFeatures[name] {
			continue
		}

		if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.AuthURL) == "" || strings.TrimSpace(cfg.TokenURL) == "" || strings.TrimSpace(cfg.APIURL) == "" {
			continue
		}

		display := strings.TrimSpace(cfg.Name)
		if display == "" {
			display = name
		}

		providers = append(providers, AuthProviderInfo{
			Name:        name,
			DisplayName: cfg.Name,
			LoginPath:   "/login/" + url.PathEscape(name),
		})
	}
	sort.Slice(providers, func(i, j int) bool {
		return strings.ToLower(providers[i].DisplayName) < strings.ToLower(providers[j].DisplayName)
	})

	c.JSON(http.StatusOK, gin.H{
		"mode":                s.cfg.Mode,
		"local_login_enabled": !s.cfg.IsCloud(),
		"providers":           providers,
	})
}

func (s *Server) ChangePassword(c *gin.Context) {
	userID, ok := s.userIDFromSession(c)
	if !ok {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	currentPassword := strings.TrimSpace(req.CurrentPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	if currentPassword == "" {
		AbortWithError(c, newValidationError("current_password", "required", "current password is required"))
		return
	}
	if newPassword == "" {
		AbortWithError(c, newValidationError("new_password", "required", "new password is required"))
		return
	}
	if currentPassword == newPassword {
		AbortWithError(c, newValidationError("new_password", "must_differ", "new password must be different"))
		return
	}
	if len(newPassword) < 8 {
		AbortWithError(c, newValidationError("new_password", "weak_password", "password must be at least 8 characters"))
		return
	}

	var user authdomain.User
	if err := s.db.WithContext(c.Request.Context()).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			AbortWithError(c, ErrUnauthorized)
			return
		}
		AbortWithError(c, err)
		return
	}
	if user.Provider != "local" {
		AbortWithError(c, ErrForbidden)
		return
	}
	if user.PasswordHash == nil || !verifyPassword(currentPassword, *user.PasswordHash) {
		AbortWithError(c, authdomain.ErrInvalidCredentials)
		return
	}

	if err := s.authsvc.ChangePassword(c.Request.Context(), userID.String(), newPassword); err != nil {
		AbortWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) Logout(c *gin.Context) {
	token, ok := s.sessions.ReadToken(c)
	if !ok {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	if err := s.authsvc.Logout(c.Request.Context(), token); err != nil {
		AbortWithError(c, err)
		return
	}

	s.sessions.Clear(c)
	c.Status(http.StatusNoContent)
}

func (s *Server) Me(c *gin.Context) {
	token, ok := s.sessions.ReadToken(c)
	if !ok {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	session, err := s.authsvc.Authenticate(c.Request.Context(), token)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	var user authdomain.User
	if err := s.db.WithContext(c.Request.Context()).First(&user, "id = ?", session.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			AbortWithError(c, ErrUnauthorized)
			return
		}
		AbortWithError(c, err)
		return
	}

	mustChangePassword := false
	passwordState := "rotated"
	if user.Provider == "local" && (user.IsDefault || user.LastPasswordChanged == nil) {
		mustChangePassword = true
		passwordState = "default"
	}

	orgIDs, err := s.loadUserOrgIDs(c.Request.Context(), user.ID)
	if err != nil {
		AbortWithError(c, err)
		return
	}
	if err := s.authsvc.UpdateSessionOrgContext(c.Request.Context(), session.ID, session.ActiveOrgID, orgIDs); err != nil {
		AbortWithError(c, err)
		return
	}
	session.OrgIDs = orgIDs

	metadata := map[string]any{
		"user_id":               user.ID.String(),
		"external_id":           user.ExternalID,
		"provider":              user.Provider,
		"display_name":          user.DisplayName,
		"email":                 user.Email,
		"is_default":            user.IsDefault,
		"last_password_changed": user.LastPasswordChanged,
		"must_change_password":  mustChangePassword,
		"password_state":        passwordState,
		"auth_provider":         user.Provider,
		"org_ids":               toOrgIDStrings(orgIDs),
	}
	if session.ActiveOrgID != nil {
		metadata["active_org_id"] = snowflake.ID(*session.ActiveOrgID).String()
	}

	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, &authdomain.SessionView{Metadata: metadata})
}

func (s *Server) Forgot(c *gin.Context) {
	AbortWithError(c, ErrServiceUnavailable)
}

func (s *Server) enrichSessionMetadata(c *gin.Context, result *authdomain.LoginResult) {
	if result == nil || result.Session == nil {
		return
	}

	rawUserID, ok := result.Session.Metadata["user_id"].(string)
	if !ok {
		return
	}

	parsedUserID, err := snowflake.ParseString(rawUserID)
	if err != nil {
		return
	}

	orgIDs, err := s.loadUserOrgIDs(c.Request.Context(), parsedUserID)
	if err != nil {
		return
	}

	if err := s.authsvc.UpdateSessionOrgContext(c.Request.Context(), result.SessionID, nil, orgIDs); err != nil {
		return
	}

	result.Session.Metadata["org_ids"] = toOrgIDStrings(orgIDs)
}
