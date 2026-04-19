package http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apikeydomain "github.com/railzwaylabs/railzway/internal/apikey/domain"
	apikeyservice "github.com/railzwaylabs/railzway/internal/apikey/service"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	entitlementdomain "github.com/railzwaylabs/railzway/internal/entitlement/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/ratelimit"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

// Handler handles API key-authenticated public API endpoints.
type Handler struct {
	apiKeys       *apikeyservice.Service
	invoices      invoicedomain.Service
	customers     customerdomain.Service
	subscriptions subscriptiondomain.Service
	usage         usagedomain.Service
	entitlement   entitlementdomain.Service
	rateLimiter   *ratelimit.Limiter
}

func NewHandler(
	apiKeys *apikeyservice.Service,
	invoices invoicedomain.Service,
	customers customerdomain.Service,
	subscriptions subscriptiondomain.Service,
	usage usagedomain.Service,
	entitlement entitlementdomain.Service,
	rateLimiter *ratelimit.Limiter,
) *Handler {
	return &Handler{
		apiKeys:       apiKeys,
		invoices:      invoices,
		customers:     customers,
		subscriptions: subscriptions,
		usage:         usage,
		entitlement:   entitlement,
		rateLimiter:   rateLimiter,
	}
}

// RegisterRoutes registers API key-authenticated routes under /api/v1.
func RegisterRoutes(r *gin.Engine, h *Handler) {
	api := r.Group("/api/v1")
	api.POST("/usage-events", h.APIKeyRequired("usage_events"), h.CreateUsageEvent)
	api.GET("/usage-events", h.APIKeyRequired("usage_events"), h.ListUsageEvents)
	api.POST("/customers", h.APIKeyRequired("customers"), h.CreateCustomer)
	api.GET("/customers", h.APIKeyRequired("customers"), h.ListCustomers)
	api.GET("/customers/:customer_id", h.APIKeyRequired("customers"), h.GetCustomer)
	api.POST("/subscriptions", h.APIKeyRequired("subscriptions"), h.CreateSubscription)
	api.GET("/subscriptions", h.APIKeyRequired("subscriptions"), h.ListSubscriptions)
	api.GET("/subscriptions/:subscription_id", h.APIKeyRequired("subscriptions"), h.GetSubscription)
	api.GET("/invoices", h.APIKeyRequired("invoices"), h.ListInvoices)
	api.GET("/invoices/:invoice_id", h.APIKeyRequired("invoices"), h.GetInvoiceByID)
	api.GET("/customers/:customer_id/entitlements/:feature_code", h.APIKeyRequired("entitlements"), h.CheckEntitlement)
}

// APIKeyRequired is a middleware that validates the API key and injects org context.
func (h *Handler) APIKeyRequired(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.apiKeys == nil {
			c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{"error": "api_key_not_configured"})
			return
		}
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_api_key"})
			return
		}
		ip := resolveClientIP(c)
		domain := resolveRequestDomain(c)
		apiKey, err := h.apiKeys.AuthorizeKey(c.Request.Context(), rawKey, apikeydomain.AuthorizeAPIKeyRequest{
			Resource: resource,
			Action:   c.Request.Method,
			IP:       ip,
			Domain:   domain,
		})
		if err != nil {
			switch {
			case errors.Is(err, apikeydomain.ErrInvalidKey):
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
			case errors.Is(err, apikeydomain.ErrKeyRevoked):
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api_key_revoked"})
			case errors.Is(err, apikeydomain.ErrKeyNotAllowed):
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api_key_not_allowed"})
			case errors.Is(err, apikeydomain.ErrScopeForbidden):
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api_key_scope_forbidden"})
			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			}
			return
		}
		orgID, err := uuid.Parse(apiKey.OrgID)
		if err != nil || orgID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
			return
		}
		if headerOrg := strings.TrimSpace(c.GetHeader("X-Org-ID")); headerOrg != "" {
			headerOrgID, parseErr := uuid.Parse(headerOrg)
			if parseErr != nil || headerOrgID == uuid.Nil || headerOrgID != orgID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "org_mismatch"})
				return
			}
		}
		ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("public_api_key_id", apiKey.ID)
		c.Set("public_org_id", apiKey.OrgID)
		c.Next()
	}
}

func extractAPIKey(c *gin.Context) string {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if raw != "" && strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	if v := strings.TrimSpace(c.GetHeader("X-API-Key")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.Query("api_key")); v != "" {
		return v
	}
	return ""
}

func resolveClientIP(c *gin.Context) string {
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if realIP := strings.TrimSpace(c.GetHeader("X-Real-IP")); realIP != "" {
		return realIP
	}
	return c.ClientIP()
}

func resolveRequestDomain(c *gin.Context) string {
	for _, header := range []string{"Origin", "Referer"} {
		raw := strings.TrimSpace(c.GetHeader(header))
		if raw == "" {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := strings.ToLower(parsed.Hostname())
		if host != "" {
			return host
		}
	}
	host := strings.ToLower(strings.TrimSpace(c.Request.Host))
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		return parts[0]
	}
	return host
}
