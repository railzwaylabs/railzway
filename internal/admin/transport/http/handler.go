package http

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminauth "github.com/railzwaylabs/railzway/internal/admin/auth"
	adminservice "github.com/railzwaylabs/railzway/internal/admin/service"
	aiassistantdomain "github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	aiworkflowdomain "github.com/railzwaylabs/railzway/internal/aiworkflow/domain"
	apikeyservice "github.com/railzwaylabs/railzway/internal/apikey/service"
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/authz"
	"github.com/railzwaylabs/railzway/internal/config"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	featureflagsvc "github.com/railzwaylabs/railzway/internal/featureflag/service"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	ledgerdomain "github.com/railzwaylabs/railzway/internal/ledger/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	referencedomain "github.com/railzwaylabs/railzway/internal/reference/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	taxdomain "github.com/railzwaylabs/railzway/internal/tax/domain"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

type Handler struct {
	summary         *adminservice.Service
	flags           *featureflagsvc.Service
	auth            *adminauth.Service
	adminAuthz      *authz.AdminAuthorizer
	audit           *auditlog.Service
	cfg             *config.Config
	apps            appsdomain.Service
	apiKeys         *apikeyservice.Service
	plans           plandomain.Service
	products        productdomain.Service
	features        featuredomain.Service
	productFeatures productfeaturedomain.Service
	customers       customerdomain.Service
	organizations   organizationdomain.Service
	subscriptions   subscriptiondomain.Service
	usage           usagedomain.Service
	invoices        invoicedomain.Service
	ledger          ledgerdomain.Service
	rating          ratingdomain.Service
	payments        paymentdomain.Service
	taxes           taxdomain.Service
	testclocks      testclockdomain.Service
	references      referencedomain.Repository
	aiAssistant     aiassistantdomain.Service
	aiWorkflow      aiworkflowdomain.Service
}

func NewHandler(
	summary *adminservice.Service,
	flags *featureflagsvc.Service,
	authSvc *adminauth.Service,
	adminAuthz *authz.AdminAuthorizer,
	auditSvc *auditlog.Service,
	cfg *config.Config,
	apps appsdomain.Service,
	apiKeys *apikeyservice.Service,
	plans plandomain.Service,
	products productdomain.Service,
	features featuredomain.Service,
	productFeatures productfeaturedomain.Service,
	customers customerdomain.Service,
	organizations organizationdomain.Service,
	subscriptions subscriptiondomain.Service,
	usage usagedomain.Service,
	invoices invoicedomain.Service,
	ledger ledgerdomain.Service,
	rating ratingdomain.Service,
	payments paymentdomain.Service,
	taxes taxdomain.Service,
	testclocks testclockdomain.Service,
	references referencedomain.Repository,
	aiAssistant aiassistantdomain.Service,
	aiWorkflow aiworkflowdomain.Service,
) *Handler {
	return &Handler{
		summary:         summary,
		flags:           flags,
		auth:            authSvc,
		adminAuthz:      adminAuthz,
		audit:           auditSvc,
		cfg:             cfg,
		apps:            apps,
		apiKeys:         apiKeys,
		plans:           plans,
		products:        products,
		features:        features,
		productFeatures: productFeatures,
		customers:       customers,
		organizations:   organizations,
		subscriptions:   subscriptions,
		usage:           usage,
		invoices:        invoices,
		ledger:          ledger,
		rating:          rating,
		payments:        payments,
		taxes:           taxes,
		testclocks:      testclocks,
		references:      references,
		aiAssistant:     aiAssistant,
		aiWorkflow:      aiWorkflow,
	}
}

func (h *Handler) DashboardSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.DashboardSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) CustomersSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.CustomersSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) PlansSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.PlansSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) SubscriptionsSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.SubscriptionsSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) UsageSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.UsageSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) RatingSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.RatingSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) InvoicesSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.InvoicesSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) PaymentsSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.PaymentsSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) TaxesSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.TaxesSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) AuditLogsSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.AuditLogsSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) SettingsSummary(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	summary, err := h.summary.SettingsSummary(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

type featureFlagListResponse struct {
	OrgID *string                   `json:"orgId,omitempty"`
	Flags []featureflagsvc.FlagView `json:"flags"`
}

func (h *Handler) ListFeatureFlags(c *gin.Context) {
	orgID := strings.TrimSpace(c.Query("org_id"))
	if orgID == "" {
		orgID = strings.TrimSpace(c.GetHeader("X-Org-ID"))
	}

	flags, err := h.flags.ListEffectiveFlags(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orgPtr *string
	if orgID != "" {
		orgPtr = &orgID
	}

	c.JSON(http.StatusOK, featureFlagListResponse{
		OrgID: orgPtr,
		Flags: flags,
	})
}

type upsertFeatureFlagRequest struct {
	OrgID   *string `json:"org_id,omitempty"`
	Key     string  `json:"key"`
	Enabled bool    `json:"enabled"`
	Rollout int     `json:"rollout"`
	ActorID string  `json:"actor_id,omitempty"`
}

type upsertFeatureFlagResponse struct {
	Status string `json:"status"`
}

func (h *Handler) UpsertFeatureFlag(c *gin.Context) {
	var payload upsertFeatureFlagRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	ctx := h.withAuditContext(c, c.Request.Context())

	actor := strings.TrimSpace(c.GetHeader("X-Actor-ID"))
	if actor == "" {
		actor = strings.TrimSpace(payload.ActorID)
	}

	if payload.OrgID == nil {
		if orgHeader := strings.TrimSpace(c.GetHeader("X-Org-ID")); orgHeader != "" {
			payload.OrgID = &orgHeader
		}
	}

	if err := h.flags.UpsertFlag(ctx, actor, payload.OrgID, payload.Key, payload.Enabled, payload.Rollout); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, upsertFeatureFlagResponse{Status: "ok"})
}

func orgIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	if val, ok := c.Get(adminauth.CtxOrgID); ok {
		switch v := val.(type) {
		case uuid.UUID:
			if v != uuid.Nil {
				return v, true
			}
		case string:
			if parsed, err := uuid.Parse(v); err == nil && parsed != uuid.Nil {
				return parsed, true
			}
		}
	}
	return orgIDFromRequest(c)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	UserID             uuid.UUID `json:"userId"`
	Email              string    `json:"email"`
	OrgID              uuid.UUID `json:"orgId"`
	OrgIDs             []string  `json:"orgIds"`
	MustChangePassword bool      `json:"mustChangePassword"`
	SessionExpiresAt   time.Time `json:"sessionExpiresAt"`
}

func (h *Handler) Login(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}

	var payload loginRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	resp, err := h.auth.Login(
		c.Request.Context(),
		payload.Email,
		payload.Password,
		c.GetHeader("User-Agent"),
		c.ClientIP(),
	)
	if err != nil {
		switch err {
		case adminauth.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		case adminauth.ErrNoOrganization:
			c.JSON(http.StatusForbidden, gin.H{"error": "no_organization"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		}
		return
	}

	setSessionCookie(c, resp.Token, resp.SessionExpiresAt, h.cfg)
	setCSRFCookie(c, newCSRFToken(), resp.SessionExpiresAt, h.cfg)
	c.JSON(http.StatusOK, loginResponse{
		UserID:             resp.UserID,
		Email:              resp.Email,
		OrgID:              resp.OrgID,
		OrgIDs:             resp.OrgIDs,
		MustChangePassword: resp.MustChangePassword,
		SessionExpiresAt:   resp.SessionExpiresAt,
	})
}

type meResponse struct {
	UserID             uuid.UUID `json:"userId"`
	OrgID              uuid.UUID `json:"orgId"`
	MustChangePassword bool      `json:"mustChangePassword"`
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, _ := orgIDFromContext(c)
	mustChange := false
	if val, ok := c.Get(adminauth.CtxMustChangePassword); ok {
		if flag, ok := val.(bool); ok {
			mustChange = flag
		}
	}
	c.JSON(http.StatusOK, meResponse{
		UserID:             userID,
		OrgID:              orgID,
		MustChangePassword: mustChange,
	})
}

type switchOrgResponse struct {
	Status string    `json:"status"`
	OrgID  uuid.UUID `json:"orgId"`
}

func (h *Handler) SwitchOrganization(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	orgID, err := uuid.Parse(strings.TrimSpace(c.Param("org_id")))
	if err != nil || orgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_organization"})
		return
	}
	cookieName := ""
	if h.cfg != nil {
		cookieName = strings.TrimSpace(h.cfg.SessionConfig.SessionCookie)
	}
	token := extractBearerToken(c.Request, cookieName)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
		return
	}
	if err := h.auth.SwitchOrganization(c.Request.Context(), token, orgID); err != nil {
		switch err {
		case adminauth.ErrInvalidOrganization:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_organization"})
		case adminauth.ErrNotOrganizationMember:
			c.JSON(http.StatusForbidden, gin.H{"error": "not_organization_member"})
		case adminauth.ErrSessionExpired, adminauth.ErrSessionRevoked, adminauth.ErrSessionNotFound:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		}
		return
	}
	c.JSON(http.StatusOK, switchOrgResponse{
		Status: "ok",
		OrgID:  orgID,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var payload changePasswordRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	if err := h.auth.ChangePassword(c.Request.Context(), userID, payload.CurrentPassword, payload.NewPassword); err != nil {
		switch err {
		case adminauth.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SkipPasswordChange(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	if h.cfg == nil || !h.cfg.AppEnv.IsDevelopment() {
		c.JSON(http.StatusForbidden, gin.H{"error": "skip_not_allowed"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.auth.SkipPasswordChange(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Logout(c *gin.Context) {
	if h.auth == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
		return
	}
	cookieName := ""
	if h.cfg != nil {
		cookieName = strings.TrimSpace(h.cfg.SessionConfig.SessionCookie)
	}
	token := extractBearerToken(c.Request, cookieName)
	if token != "" {
		_ = h.auth.RevokeSession(c.Request.Context(), token)
	}
	clearSessionCookie(c, h.cfg)
	clearCSRFCookie(c, h.cfg)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func userIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	if val, ok := c.Get(adminauth.CtxUserID); ok {
		switch v := val.(type) {
		case uuid.UUID:
			if v != uuid.Nil {
				return v, true
			}
		case string:
			if parsed, err := uuid.Parse(v); err == nil && parsed != uuid.Nil {
				return parsed, true
			}
		}
	}
	return uuid.Nil, false
}

func roleFromContext(c *gin.Context) (string, bool) {
	if val, ok := c.Get(adminauth.CtxRole); ok {
		if role, ok := val.(string); ok && strings.TrimSpace(role) != "" {
			return role, true
		}
	}
	return "", false
}

func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.auth == nil {
			c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
			return
		}
		cookieName := ""
		if h.cfg != nil {
			cookieName = strings.TrimSpace(h.cfg.SessionConfig.SessionCookie)
		}
		token := extractBearerToken(c.Request, cookieName)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
			return
		}
		session, err := h.auth.Authenticate(c.Request.Context(), token)
		if err != nil {
			switch err {
			case adminauth.ErrSessionExpired, adminauth.ErrSessionRevoked, adminauth.ErrSessionNotFound:
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			}
			return
		}

		// Align active org with request header when provided.
		if requestedOrgID, ok := orgIDFromRequest(c); ok && requestedOrgID != session.OrgID {
			if err := h.auth.SwitchOrganization(c.Request.Context(), token, requestedOrgID); err != nil {
				switch err {
				case adminauth.ErrInvalidOrganization:
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_organization"})
				case adminauth.ErrNotOrganizationMember:
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not_organization_member"})
				case adminauth.ErrSessionExpired, adminauth.ErrSessionRevoked, adminauth.ErrSessionNotFound:
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
				default:
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
				}
				return
			}
			session.OrgID = requestedOrgID
		}

		c.Set(adminauth.CtxUserID, session.UserID)
		c.Set(adminauth.CtxOrgID, session.OrgID)
		c.Set(adminauth.CtxMustChangePassword, session.MustChangePassword)

		ensureCSRFCookie(c, h.cfg)

		c.Next()
	}
}

func (h *Handler) AuthorizeAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.adminAuthz == nil {
			c.Next()
			return
		}
		userID, ok := userIDFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		orgID, ok := orgIDFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
			return
		}
		role, ok := roleFromContext(c)
		if !ok {
			if h.auth == nil {
				c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{"error": "auth_not_configured"})
				return
			}
			var err error
			role, err = h.auth.GetMemberRole(c.Request.Context(), userID, orgID)
			if err != nil {
				switch err {
				case adminauth.ErrNotOrganizationMember:
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not_organization_member"})
				default:
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
				}
				return
			}
			c.Set(adminauth.CtxRole, role)
		}
		path := strings.TrimSpace(c.FullPath())
		if path == "" {
			path = c.Request.URL.Path
		}
		allowed, err := h.adminAuthz.Enforce(role, path, c.Request.Method)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func (h *Handler) RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}
		expected := csrfTokenFromCookie(c, h.cfg)
		provided := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if expected == "" || provided == "" || expected != provided {
			log.Printf("CSRF MISMATCH: expected='%s' provided='%s'", expected, provided)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf_invalid"})
			return
		}
		c.Next()
	}
}

func (h *Handler) RequireOrgHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := orgIDFromRequest(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing_org_id"})
			return
		}
		c.Set(adminauth.CtxOrgID, orgID)
		c.Next()
	}
}

func orgIDFromRequest(c *gin.Context) (uuid.UUID, bool) {
	raw := strings.TrimSpace(c.GetHeader("X-Org-ID"))
	if raw == "" {
		raw = strings.TrimSpace(c.Query("org_id"))
	}
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

func requireOrgParam(c *gin.Context, param string) (uuid.UUID, bool) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return uuid.Nil, false
	}
	raw := strings.TrimSpace(c.Param(param))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return uuid.Nil, false
	}
	paramID, err := uuid.Parse(raw)
	if err != nil || paramID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return uuid.Nil, false
	}
	if paramID != orgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id_mismatch"})
		return uuid.Nil, false
	}
	return orgID, true
}

func extractBearerToken(r *http.Request, cookieName string) string {
	if strings.TrimSpace(cookieName) == "" {
		cookieName = "rz_admin_session"
	}
	if cookie, err := r.Cookie(cookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token
		}
	}
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw != "" && strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	return strings.TrimSpace(r.Header.Get("X-Admin-Token"))
}

func setSessionCookie(c *gin.Context, token string, expiresAt time.Time, cfg *config.Config) {
	secure := cfg != nil && cfg.AppEnv.IsProduction()
	cookieName := "rz_admin_session"
	if cfg != nil && strings.TrimSpace(cfg.SessionConfig.SessionCookie) != "" {
		cookieName = strings.TrimSpace(cfg.SessionConfig.SessionCookie)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(c *gin.Context, cfg *config.Config) {
	secure := cfg != nil && cfg.AppEnv.IsProduction()
	cookieName := "rz_admin_session"
	if cfg != nil && strings.TrimSpace(cfg.SessionConfig.SessionCookie) != "" {
		cookieName = strings.TrimSpace(cfg.SessionConfig.SessionCookie)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

const defaultCSRFCookieName = "rz_admin_csrf"

func csrfCookieName(cfg *config.Config) string {
	if cfg == nil || strings.TrimSpace(cfg.SessionConfig.SessionCookie) == "" {
		return defaultCSRFCookieName
	}
	return strings.TrimSpace(cfg.SessionConfig.SessionCookie) + "_csrf"
}

func newCSRFToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func csrfTokenFromCookie(c *gin.Context, cfg *config.Config) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if cookie, err := c.Request.Cookie(csrfCookieName(cfg)); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func ensureCSRFCookie(c *gin.Context, cfg *config.Config) {
	if csrfTokenFromCookie(c, cfg) != "" {
		return
	}
	ttlHours := 24
	if cfg != nil && cfg.SessionConfig.SessionTTLHours > 0 {
		ttlHours = cfg.SessionConfig.SessionTTLHours
	}
	setCSRFCookie(c, newCSRFToken(), time.Now().UTC().Add(time.Duration(ttlHours)*time.Hour), cfg)
}

func setCSRFCookie(c *gin.Context, token string, expiresAt time.Time, cfg *config.Config) {
	if token == "" {
		return
	}
	secure := cfg != nil && cfg.AppEnv.IsProduction()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName(cfg),
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCSRFCookie(c *gin.Context, cfg *config.Config) {
	secure := cfg != nil && cfg.AppEnv.IsProduction()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName(cfg),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
