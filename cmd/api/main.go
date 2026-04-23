package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	publicapi "github.com/railzwaylabs/railzway/internal/api"
	apihandler "github.com/railzwaylabs/railzway/internal/api/transport/http"
	apikeyrepo "github.com/railzwaylabs/railzway/internal/apikey/repository"
	apikeyservice "github.com/railzwaylabs/railzway/internal/apikey/service"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/cache"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/coupon"
	"github.com/railzwaylabs/railzway/internal/customer"
	"github.com/railzwaylabs/railzway/internal/db"
	"github.com/railzwaylabs/railzway/internal/entitlement"
	"github.com/railzwaylabs/railzway/internal/feature"
	"github.com/railzwaylabs/railzway/internal/httpmiddleware"
	"github.com/railzwaylabs/railzway/internal/invoice"
	"github.com/railzwaylabs/railzway/internal/plan"
	"github.com/railzwaylabs/railzway/internal/planfeature"
	"github.com/railzwaylabs/railzway/internal/product"
	"github.com/railzwaylabs/railzway/internal/productfeature"
	"github.com/railzwaylabs/railzway/internal/ratelimit"
	"github.com/railzwaylabs/railzway/internal/redis"
	subscriptionrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	subscriptionservice "github.com/railzwaylabs/railzway/internal/subscription/service"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	"github.com/railzwaylabs/railzway/internal/usage"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.RegisterFor("api"),
			newLogger,
			fx.Annotate(newRouter, fx.ParamTags("", "", "", "")),
		),
		db.Module,
		customer.Module,
		usage.Module,
		plan.Module,
		planfeature.Module,
		product.Module,
		feature.Module,
		productfeature.Module,
		entitlement.Module,
		coupon.Module,
		invoice.Module,
		cache.Register(),
		redis.Register(),
		ratelimit.Module,
		fx.Provide(subscriptionrepo.NewRepository, subscriptionservice.NewService),
		fx.Provide(apikeyrepo.NewRepository, apikeyservice.NewService),
		fx.Provide(auditlog.NewService),
		publicapi.Module,
		fx.Invoke(registerLoggerLifecycle),
		fx.Invoke(registerTelemetryLifecycle),
		fx.Invoke(telemetry.StartProfiler(6060)),
		fx.Invoke(startHTTPServer),
	)

	app.Run()
}

func newLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.AppEnv.IsProduction() {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func registerLoggerLifecycle(lc fx.Lifecycle, logger *zap.Logger) {
	zap.ReplaceGlobals(logger)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return logger.Sync()
		},
	})
}

func registerTelemetryLifecycle(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) {
	serviceName := "railzway-api"
	if cfg != nil && strings.TrimSpace(cfg.AppName) != "" {
		serviceName = strings.TrimSpace(cfg.AppName) + "-api"
	}

	var shutdown func(context.Context) error = func(context.Context) error { return nil }

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			traceShutdown, enabled, err := telemetry.InitTracing(ctx, serviceName, logger)
			if err != nil {
				return err
			}

			shutdown = traceShutdown
			if !enabled && logger != nil {
				logger.Info("opentelemetry tracing disabled", zap.String("service", serviceName))
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return shutdown(ctx)
		},
	})
}

func newRouter(cfg *config.Config, apiHandler *apihandler.Handler, logger *zap.Logger, dbConn *gorm.DB) *gin.Engine {
	if cfg.AppEnv.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpmiddleware.SecurityHeadersWithCSP(cfg, cfg.CSPExtraDirectives, httpmiddleware.CSPProfilePublic))
	router.Use(httpmiddleware.ZapRequestLogger(logger))
	router.Use(httpmiddleware.RequireTLS(cfg))

	registerHealthRoutes(router, dbConn)
	apihandler.RegisterRoutes(router, apiHandler)

	return router
}

func startHTTPServer(lc fx.Lifecycle, cfg *config.Config, router *gin.Engine, logger *zap.Logger) {
	port := cfg.AppPort
	if port == 0 {
		port = 8080
	}
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	if cfg.AppEnv.IsProduction() && cfg.AppTLSMode.IsDisabled() {
		logger.Fatal("production requires TLS", zap.String("mode", string(cfg.AppTLSMode)))
	}
	if cfg.AppTLSMode.IsDirect() && (strings.TrimSpace(cfg.AppTLSCertFile) == "" || strings.TrimSpace(cfg.AppTLSKeyFile) == "") {
		logger.Fatal("tls cert/key required for direct TLS mode")
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("api server started", zap.String("addr", addr), zap.String("tls_mode", string(cfg.AppTLSMode)))
				var err error
				if cfg.AppTLSMode.IsDirect() {
					err = server.ListenAndServeTLS(cfg.AppTLSCertFile, cfg.AppTLSKeyFile)
				} else {
					err = server.ListenAndServe()
				}
				if err != nil && err != http.ErrServerClosed {
					logger.Error("api server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("api server stopping", zap.String("addr", addr))
			return server.Shutdown(ctx)
		},
	})
}

func registerHealthRoutes(router *gin.Engine, dbConn *gorm.DB) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		type component struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}

		results := make([]component, 0, 1)
		ready := true
		var mu sync.Mutex

		record := func(name string, err error) {
			status := "ok"
			errMsg := ""
			if err != nil {
				status = "error"
				errMsg = err.Error()
			}
			mu.Lock()
			if err != nil {
				ready = false
			}
			results = append(results, component{Name: name, Status: status, Error: errMsg})
			mu.Unlock()
		}

		group, groupCtx := errgroup.WithContext(ctx)
		if dbConn != nil {
			group.Go(func() error {
				sqlDB, err := dbConn.DB()
				if err != nil {
					record("db", err)
					return err
				}
				err = sqlDB.PingContext(groupCtx)
				record("db", err)
				return err
			})
		}

		_ = group.Wait()
		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"status": map[bool]string{true: "ok", false: "error"}[ready], "components": results})
	})
}
