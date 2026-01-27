package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/license"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLicenseEnforcement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// 1. Generate keys
	privStr, pubStr, err := license.GenerateKeyPair()
	require.NoError(t, err)

	// 2. Setup License Service
	licenseSvc, err := license.NewService(logger, config.Config{
		License: config.LicenseConfig{PublicKey: pubStr},
	})
	require.NoError(t, err)

	// 3. Setup Server
	srv := &Server{
		cfg:        config.Config{Mode: config.ModeOSS},
		licenseSvc: licenseSvc,
	}

	// 4. Setup Router with Middleware
	router := gin.New()
	router.Use(srv.LicenseContext())
	router.GET("/protected", srv.RequireCapability("sso"), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("No License -> Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
	})

	t.Run("Valid License -> Allowed", func(t *testing.T) {
		// Generate valid license
		payload := license.Payload{
			OrgID:     "acme",
			Features:  []string{"sso"},
			ExpiresAt: time.Now().Add(time.Hour),
		}
		licBytes, err := license.SignLicense(privStr, payload)
		require.NoError(t, err)

		// Load license
		err = licenseSvc.LoadLicense(string(licBytes))
		require.NoError(t, err)

		// Request
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, "ok", resp.Body.String())
	})

	t.Run("Valid License BUT Missing Capability -> Forbidden", func(t *testing.T) {
		// Generate license without SSO
		payload := license.Payload{
			OrgID:     "basic-corp",
			Features:  []string{"audit_log"}, // Has audit, but not SSO
			ExpiresAt: time.Now().Add(time.Hour),
		}
		licBytes, err := license.SignLicense(privStr, payload)
		require.NoError(t, err)

		// Load license
		err = licenseSvc.LoadLicense(string(licBytes))
		require.NoError(t, err)

		// Request
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
	})
}
