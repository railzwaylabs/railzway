package http

import (
	"net/http"
	"testing"

	"github.com/railzwaylabs/railzway/internal/config"
)

func TestSessionCookieSameSiteDefaultsToLax(t *testing.T) {
	if got := sessionCookieSameSite(nil); got != http.SameSiteLaxMode {
		t.Fatalf("expected lax by default, got %v", got)
	}
}

func TestSessionCookieSameSiteRespectsConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Session.CookieSameSite = "strict"
	if got := sessionCookieSameSite(cfg); got != http.SameSiteStrictMode {
		t.Fatalf("expected strict same-site, got %v", got)
	}

	cfg.Session.CookieSameSite = "none"
	if got := sessionCookieSameSite(cfg); got != http.SameSiteNoneMode {
		t.Fatalf("expected none same-site, got %v", got)
	}
}

func TestSessionCookieDomainTrimsWhitespace(t *testing.T) {
	cfg := &config.Config{}
	cfg.Session.CookieDomain = "  .yourdomain.com  "
	if got := sessionCookieDomain(cfg); got != ".yourdomain.com" {
		t.Fatalf("expected trimmed cookie domain, got %q", got)
	}
}
