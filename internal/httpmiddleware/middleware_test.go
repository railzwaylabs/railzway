package httpmiddleware

import (
	"context"
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
	cfg.App.Env = config.AppEnvProduction
	cfg.App.TLSMode = config.TLSModeDirect

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
	cfg.App.Env = config.AppEnvDevelopment
	cfg.App.TLSMode = config.TLSModeDisabled

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
	cfg.App.Env = config.AppEnvProduction
	cfg.App.TLSMode = config.TLSModeProxy

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
	cfg.App.Env = config.AppEnvProduction
	cfg.App.TLSMode = config.TLSModeProxy

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

func TestBrowserCORSAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.Browser.AllowedOrigins = []string{"https://manage.yourdomain.com"}

	r.Use(BrowserCORS(cfg))
	r.OPTIONS("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://manage.yourdomain.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "X-CSRF-Token, X-Org-ID")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://manage.yourdomain.com" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected allow credentials header, got %q", got)
	}
}

func TestBrowserCORSRejectsDisallowedPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.Browser.AllowedOrigins = []string{"https://manage.yourdomain.com"}

	r.Use(BrowserCORS(cfg))
	r.OPTIONS("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow origin header, got %q", got)
	}
}

func TestZapRequestLoggerEmitsLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	r.Use(Correlation())
	r.Use(ZapRequestLogger(logger))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if logs.Len() == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := logs.All()[0]
	fields := entry.ContextMap()
	if fields["request_id"] == "" {
		t.Fatalf("expected request_id field in log")
	}
	if fields["correlation_id"] == "" {
		t.Fatalf("expected correlation_id field in log")
	}
	if fields["trace_id"] == "" {
		t.Fatalf("expected trace_id field in log")
	}
}

func TestCorrelationAddsHeadersAndContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Correlation())
	r.GET("/", func(c *gin.Context) {
		requestID := RequestIDFromContext(c.Request.Context())
		correlationID := CorrelationIDFromContext(c.Request.Context())
		traceID := TraceIDFromContext(c.Request.Context())
		if requestID == "" || correlationID == "" || traceID == "" {
			t.Fatalf("expected correlation IDs in request context")
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID response header")
	}
	if w.Header().Get("X-Correlation-ID") == "" {
		t.Fatalf("expected X-Correlation-ID response header")
	}
	if w.Header().Get("X-Trace-ID") == "" {
		t.Fatalf("expected X-Trace-ID response header")
	}
}

func TestCorrelationRespectsIncomingHeaders(t *testing.T) {
	ctx := context.Background()
	rid := "req-123"
	cid := "corr-123"
	tid := "0123456789abcdef0123456789abcdef"
	ctx = context.WithValue(ctx, requestIDContextKey, rid)
	ctx = context.WithValue(ctx, correlationIDContextKey, cid)
	ctx = context.WithValue(ctx, traceIDContextKey, tid)

	if got := RequestIDFromContext(ctx); got != rid {
		t.Fatalf("expected request id %q, got %q", rid, got)
	}
	if got := CorrelationIDFromContext(ctx); got != cid {
		t.Fatalf("expected correlation id %q, got %q", cid, got)
	}
	if got := TraceIDFromContext(ctx); got != tid {
		t.Fatalf("expected trace id %q, got %q", tid, got)
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
