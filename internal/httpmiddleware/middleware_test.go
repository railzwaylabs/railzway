package httpmiddleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestSecurityHeadersAddsCSPAndHSTS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.AppEnv = config.AppEnvProduction
	cfg.AppTLSMode = config.TLSModeDirect

	r.Use(SecurityHeaders(cfg, CSPProfileAdmin))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("expected CSP header to be set")
	}
	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("expected HSTS header in production direct mode")
	}
}

func TestSecurityHeadersWithExtraDirectives(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.AppEnv = config.AppEnvDevelopment
	cfg.AppTLSMode = config.TLSModeDisabled

	extra := "connect-src 'self' https://api.example.com"
	r.Use(SecurityHeadersWithCSP(cfg, extra, CSPProfileAdmin))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatalf("expected CSP header to be set")
	}
	if !contains(csp, extra) {
		t.Fatalf("expected CSP to contain extra directives")
	}
}

func TestRequireTLSBlocksWhenNotForwarded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.AppEnv = config.AppEnvProduction
	cfg.AppTLSMode = config.TLSModeProxy

	r.Use(RequireTLS(cfg))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUpgradeRequired {
		t.Fatalf("expected status %d, got %d", http.StatusUpgradeRequired, w.Code)
	}
}

func TestRequireTLSPassesWhenForwardedHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.AppEnv = config.AppEnvProduction
	cfg.AppTLSMode = config.TLSModeProxy

	r.Use(RequireTLS(cfg))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestZapRequestLoggerEmitsLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	r.Use(ZapRequestLogger(logger))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if logs.Len() == 0 {
		t.Fatalf("expected at least one log entry")
	}
}

func TestDefaultCSPExtraCheckout(t *testing.T) {
	extra := DefaultCSPExtra(CSPProfileCheckout)
	if extra == "" {
		t.Fatalf("expected default checkout CSP extra directives")
	}
	if !contains(extra, "js.stripe.com") {
		t.Fatalf("expected default checkout CSP to include stripe")
	}
}

func contains(value, part string) bool {
	return strings.Contains(value, part)
}
