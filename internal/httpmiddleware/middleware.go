package httpmiddleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/zap"
)

func RequireTLS(cfg *config.Config) gin.HandlerFunc {
	if !cfg.AppEnv.IsProduction() || !cfg.AppTLSMode.IsProxy() {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if isForwardedHTTPS(c.Request) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUpgradeRequired, gin.H{"error": "tls_required"})
	}
}

type CSPProfile string

const (
	CSPProfileAdmin    CSPProfile = "admin"
	CSPProfileCustomer CSPProfile = "customer"
	CSPProfileCheckout CSPProfile = "checkout"
	CSPProfilePublic   CSPProfile = "public"
)

func DefaultCSPExtra(profile CSPProfile) string {
	switch profile {
	case CSPProfileCheckout:
		return strings.Join([]string{
			"script-src 'self' https://js.stripe.com",
			"frame-src https://js.stripe.com",
			"connect-src 'self' https://api.stripe.com",
		}, "; ")
	default:
		return ""
	}
}

func SecurityHeaders(cfg *config.Config, profile CSPProfile) gin.HandlerFunc {
	return SecurityHeadersWithCSP(cfg, "", profile)
}

func mergeCSP(base, extra string) string {
	parse := func(csp string, order *[]string, values map[string][]string, seen map[string]map[string]struct{}) {
		for _, chunk := range strings.Split(csp, ";") {
			directive := strings.TrimSpace(chunk)
			if directive == "" {
				continue
			}
			parts := strings.Fields(directive)
			if len(parts) == 0 {
				continue
			}
			name := parts[0]
			if _, ok := seen[name]; !ok {
				seen[name] = map[string]struct{}{}
				*order = append(*order, name)
			}
			for _, value := range parts[1:] {
				if _, ok := seen[name][value]; ok {
					continue
				}
				seen[name][value] = struct{}{}
				values[name] = append(values[name], value)
			}
		}
	}

	order := []string{}
	values := map[string][]string{}
	seen := map[string]map[string]struct{}{}
	parse(base, &order, values, seen)
	parse(extra, &order, values, seen)

	var parts []string
	for _, name := range order {
		valuesList := values[name]
		if len(valuesList) == 0 {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, name+" "+strings.Join(valuesList, " "))
	}
	return strings.Join(parts, "; ")
}

func SecurityHeadersWithCSP(cfg *config.Config, extraDirectives string, profile CSPProfile) gin.HandlerFunc {
	csp := strings.Join([]string{
		"default-src 'self'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"object-src 'none'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self' data:",
		"connect-src 'self'",
	}, "; ")
	extra := strings.TrimSpace(extraDirectives)
	defaultExtra := strings.TrimSpace(DefaultCSPExtra(profile))
	mergedExtra := strings.TrimSpace(strings.Join([]string{defaultExtra, extra}, "; "))
	if mergedExtra != "" {
		csp = mergeCSP(csp, mergedExtra)
	}

	enableHSTS := cfg != nil && cfg.AppEnv.IsProduction() &&
		(cfg.AppTLSMode.IsDirect() || cfg.AppTLSMode.IsProxy())

	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", csp)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		if enableHSTS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func ZapRequestLogger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.L()
	}
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}
		latency := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()

		clientIP := resolveClientIP(c)
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.Int("bytes", size),
		}
		if v := c.GetHeader("X-Trace-Id"); v != "" {
			fields = append(fields, zap.String("trace_id", v))
		}
		if v := c.GetHeader("X-Request-Id"); v != "" {
			fields = append(fields, zap.String("request_id", v))
		}
		if v := c.GetHeader("X-Org-Id"); v != "" {
			fields = append(fields, zap.String("org_id", v))
		}
		if v := c.GetHeader("X-Customer-Id"); v != "" {
			fields = append(fields, zap.String("customer_id", v))
		}
		if v := c.GetHeader("X-Correlation-Id"); v != "" {
			fields = append(fields, zap.String("correlation_id", v))
		}
		if v := c.GetHeader("X-Forwarded-For"); v != "" {
			fields = append(fields, zap.String("forwarded_for", v))
		}
		if v := c.GetHeader("X-Real-IP"); v != "" {
			fields = append(fields, zap.String("real_ip", v))
		}

		if len(c.Errors) > 0 {
			logger.Error("http request", append(fields, zap.String("errors", c.Errors.String()))...)
			return
		}

		logger.Info("http request", fields...)
	}
}

func isForwardedHTTPS(r *http.Request) bool {
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if proto == "https" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-SSL")), "on") {
		return true
	}
	forwarded := strings.ToLower(r.Header.Get("Forwarded"))
	return strings.Contains(forwarded, "proto=https")
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
