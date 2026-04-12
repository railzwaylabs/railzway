package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type installAppPayload struct {
	AppID       string          `json:"app_id"`
	AuthMethod  string          `json:"auth_method"`
	Config      json.RawMessage `json:"config"`
	Credentials json.RawMessage `json:"credentials"`
}

type updateInstallationPayload struct {
	Status      *string         `json:"status"`
	AuthMethod  *string         `json:"auth_method"`
	Config      json.RawMessage `json:"config"`
	Credentials json.RawMessage `json:"credentials"`
}

func (h *Handler) ListAppsCatalog(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	resp, err := h.apps.ListCatalog(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListAppInstallations(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	resp, err := h.apps.ListInstallations(ctx)
	if err != nil {
		writeAppsError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) InstallApp(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	var payload installAppPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	resp, err := h.apps.InstallApp(ctx, appsdomain.InstallAppRequest{
		AppID:       payload.AppID,
		AuthMethod:  payload.AuthMethod,
		Config:      payload.Config,
		Credentials: payload.Credentials,
	})
	if err != nil {
		writeAppsError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateAppInstallation(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	installID := c.Param("installation_id")
	var payload updateInstallationPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	resp, err := h.apps.UpdateInstallation(ctx, installID, appsdomain.UpdateInstallationRequest{
		Status:      payload.Status,
		AuthMethod:  payload.AuthMethod,
		Config:      payload.Config,
		Credentials: payload.Credentials,
	})
	if err != nil {
		writeAppsError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type stripeOAuthState struct {
	OrgID    string `json:"org_id"`
	UserID   string `json:"user_id"`
	IssuedAt int64  `json:"iat"`
	Nonce    string `json:"nonce"`
}

type stripeTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Livemode         bool   `json:"livemode"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	StripeUserID     string `json:"stripe_user_id"`
	Scope            string `json:"scope"`
	PublishableKey   string `json:"stripe_publishable_key"`
	StripeAccountID  string `json:"stripe_account_id"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (h *Handler) StartStripeOAuth(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.cfg == nil || strings.TrimSpace(h.cfg.StripeConnectClientID) == "" || strings.TrimSpace(h.cfg.StripeConnectRedirectURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stripe_not_configured"})
		return
	}
	secret := ""
	if h.cfg != nil {
		secret = strings.TrimSpace(h.cfg.SessionConfig.SessionSecret)
	}
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_secret_required"})
		return
	}

	state, err := buildStripeState(stripeOAuthState{
		OrgID:    orgID.String(),
		UserID:   userID.String(),
		IssuedAt: time.Now().UTC().Unix(),
		Nonce:    randomString(12),
	}, secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", strings.TrimSpace(h.cfg.StripeConnectClientID))
	query.Set("scope", "read_write")
	query.Set("state", state)
	query.Set("redirect_uri", strings.TrimSpace(h.cfg.StripeConnectRedirectURL))
	urlStr := "https://connect.stripe.com/oauth/authorize?" + query.Encode()

	c.JSON(http.StatusOK, gin.H{"url": urlStr})
}

func (h *Handler) StripeOAuthCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_oauth_response"})
		return
	}
	if h.cfg == nil || strings.TrimSpace(h.cfg.StripeConnectSecret) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stripe_not_configured"})
		return
	}
	secret := ""
	if h.cfg != nil {
		secret = strings.TrimSpace(h.cfg.SessionConfig.SessionSecret)
	}
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_secret_required"})
		return
	}
	payload, err := parseStripeState(state, secret, 10*time.Minute)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
		return
	}
	token, err := exchangeStripeCode(code, strings.TrimSpace(h.cfg.StripeConnectSecret))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth_exchange_failed"})
		return
	}

	credentials := map[string]interface{}{
		"access_token":       token.AccessToken,
		"refresh_token":      token.RefreshToken,
		"token_type":         token.TokenType,
		"stripe_user_id":     token.StripeUserID,
		"stripe_account_id":  token.StripeAccountID,
		"stripe_publishable": token.PublishableKey,
		"livemode":           token.Livemode,
		"scope":              token.Scope,
	}
	rawCreds, _ := json.Marshal(credentials)

	orgUUID, err := uuid.Parse(payload.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgUUID)
	_, err = h.apps.InstallApp(ctx, appsdomain.InstallAppRequest{
		AppID:       "payment.stripe",
		AuthMethod:  "oauth2",
		Credentials: rawCreds,
	})
	if err != nil {
		writeAppsError(c, err)
		return
	}

	redirectURL := buildRedirectURL(c, payload.OrgID, "stripe_connected")
	c.Redirect(http.StatusFound, redirectURL)
}

func exchangeStripeCode(code, clientSecret string) (*stripeTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_secret", clientSecret)

	resp, err := http.PostForm("https://connect.stripe.com/oauth/token", form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var token stripeTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || token.Error != "" {
		return nil, errors.New("stripe_oauth_failed")
	}
	return &token, nil
}

func buildStripeState(payload stripeOAuthState, secret string) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	signature := signState(body, secret)
	return body + "." + signature, nil
}

func parseStripeState(state, secret string, maxAge time.Duration) (stripeOAuthState, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return stripeOAuthState{}, appsdomain.ErrInvalidApp
	}
	body := parts[0]
	sig := parts[1]
	expected := signState(body, secret)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return stripeOAuthState{}, appsdomain.ErrInvalidApp
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return stripeOAuthState{}, err
	}
	var payload stripeOAuthState
	if err := json.Unmarshal(raw, &payload); err != nil {
		return stripeOAuthState{}, err
	}
	if payload.IssuedAt == 0 {
		return stripeOAuthState{}, appsdomain.ErrInvalidApp
	}
	issued := time.Unix(payload.IssuedAt, 0)
	if time.Since(issued) > maxAge {
		return stripeOAuthState{}, appsdomain.ErrInvalidApp
	}
	return payload, nil
}

func signState(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomString(length int) string {
	if length <= 0 {
		return ""
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func buildRedirectURL(c *gin.Context, orgID, status string) string {
	scheme := "http"
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	if host == "" {
		host = "localhost:8080"
	}
	return scheme + "://" + host + "/organizations/" + orgID + "/apps?status=" + url.QueryEscape(status)
}
