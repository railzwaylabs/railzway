package session

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smallbiznis/railzway/internal/config"
)

const DefaultCookieName = "_sid"

// Manager manages auth session cookies.
type Manager struct {
	cookieName string
	secure     bool
}

func NewManager(cfg config.Config) *Manager {
	return &Manager{
		cookieName: DefaultCookieName,
		secure:     cfg.AuthCookieSecure,
	}
}

func (m *Manager) CookieName() string {
	return m.cookieName
}

func (m *Manager) ReadToken(c *gin.Context) (string, bool) {
	token, err := c.Cookie(m.cookieName)
	if err != nil {
		return "", false
	}
	if strings.TrimSpace(token) == "" {
		return "", false
	}
	return token, true
}

func (m *Manager) Set(c *gin.Context, value string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(m.cookieName, value, maxAge, "/", "", m.secure, true)
}

func (m *Manager) Clear(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(m.cookieName, "", -1, "/", "", m.secure, true)
}
