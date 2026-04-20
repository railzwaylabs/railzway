package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/admin"
	adminhandler "github.com/railzwaylabs/railzway/internal/admin/transport/http"
	aimodule "github.com/railzwaylabs/railzway/internal/ai"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	"github.com/railzwaylabs/railzway/internal/cache"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/coupon"
	"github.com/railzwaylabs/railzway/internal/customer"
	"github.com/railzwaylabs/railzway/internal/db"
	"github.com/railzwaylabs/railzway/internal/feature"
	"github.com/railzwaylabs/railzway/internal/featureflag"
	"github.com/railzwaylabs/railzway/internal/httpmiddleware"
	"github.com/railzwaylabs/railzway/internal/invoice"
	"github.com/railzwaylabs/railzway/internal/ledger"
	"github.com/railzwaylabs/railzway/internal/organization"
	"github.com/railzwaylabs/railzway/internal/payment"
	"github.com/railzwaylabs/railzway/internal/plan"
	"github.com/railzwaylabs/railzway/internal/product"
	"github.com/railzwaylabs/railzway/internal/productfeature"
	"github.com/railzwaylabs/railzway/internal/rating"
	redisstore "github.com/railzwaylabs/railzway/internal/redis"
	"github.com/railzwaylabs/railzway/internal/reference"
	"github.com/railzwaylabs/railzway/internal/subscription"
	"github.com/railzwaylabs/railzway/internal/tax"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	"github.com/railzwaylabs/railzway/internal/testclock"
	"github.com/railzwaylabs/railzway/internal/usage"
	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Register,
			newLogger,
			fx.Annotate(
				newRouter,
				fx.ParamTags(
					"",
					"",
					"",
					"",
					fmt.Sprintf(`name:"%s"`, cache.ClientName),
					fmt.Sprintf(`name:"%s"`, redisstore.StoreClientName),
				),
			),
		),
		db.Module,
		clock.Module,
		testclock.Module,
		cache.Register(),
		redisstore.Register(),
		bootstrap.Module,
		customer.Module,
		plan.Module,
		product.Module,
		feature.Module,
		productfeature.Module,
		subscription.Module,
		usage.Module,
		coupon.Module,
		invoice.Module,
		ledger.Module,
		organization.Module,
		payment.Module,
		rating.Module,
		reference.Module,
		featureflag.Module,
		tax.Module,
		aimodule.Module,
		admin.Module,
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
	serviceName := "railzway-admin"
	if cfg != nil && strings.TrimSpace(cfg.AppName) != "" {
		serviceName = strings.TrimSpace(cfg.AppName) + "-admin"
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

func newRouter(
	cfg *config.Config,
	adminHandler *adminhandler.Handler,
	logger *zap.Logger,
	dbConn *gorm.DB,
	cacheClient *redisv9.Client,
	storeClient *redisv9.Client,
) *gin.Engine {
	if cfg.AppEnv.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpmiddleware.SecurityHeadersWithCSP(cfg, cfg.CSPExtraDirectives, httpmiddleware.CSPProfileAdmin))
	router.Use(httpmiddleware.Correlation())
	router.Use(httpmiddleware.PrometheusMetrics("admin"))
	router.Use(httpmiddleware.ZapRequestLogger(logger))
	router.Use(httpmiddleware.RequireTLS(cfg))

	registerHealthRoutes(router, dbConn, cacheClient, storeClient)
	router.GET("/metrics", gin.WrapH(telemetry.MetricsHandler()))
	adminhandler.RegisterRoutes(router, adminHandler)

	distDir := filepath.Join("apps", "admin", "dist")
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/admin") {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not_found"})
				return
			}
			clean := filepath.Clean(path)
			if strings.Contains(clean, "..") {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_path"})
				return
			}
			if clean == "." || clean == string(os.PathSeparator) {
				c.File(filepath.Join(distDir, "index.html"))
				return
			}
			clean = strings.TrimPrefix(clean, string(os.PathSeparator))
			filePath := filepath.Join(distDir, clean)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				return
			}
			c.File(filepath.Join(distDir, "index.html"))
		})
		logger.Info("admin ui static serving enabled", zap.String("dir", distDir))
	} else {
		logger.Warn("admin ui dist not found, skipping static serve", zap.String("dir", distDir))
	}

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
	if strings.TrimSpace(cfg.SessionConfig.SessionSecret) == "" {
		logger.Fatal("SESSION_SECRET is required")
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("admin server started", zap.String("addr", addr), zap.String("tls_mode", string(cfg.AppTLSMode)))
				var err error
				if cfg.AppTLSMode.IsDirect() {
					err = server.ListenAndServeTLS(cfg.AppTLSCertFile, cfg.AppTLSKeyFile)
				} else {
					err = server.ListenAndServe()
				}
				if err != nil && err != http.ErrServerClosed {
					logger.Error("admin server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("admin server stopping", zap.String("addr", addr))
			return server.Shutdown(ctx)
		},
	})
}

func registerHealthRoutes(router *gin.Engine, dbConn *gorm.DB, cacheClient *redisv9.Client, storeClient *redisv9.Client) {
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

		results := make([]component, 0, 3)
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

		if cacheClient != nil {
			group.Go(func() error {
				err := cacheClient.Ping(groupCtx).Err()
				record("cache", err)
				return err
			})
		}

		if storeClient != nil {
			group.Go(func() error {
				err := storeClient.Ping(groupCtx).Err()
				record("redis", err)
				return err
			})
		}

		_ = group.Wait()

		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{
			"status":     map[bool]string{true: "ready", false: "not_ready"}[ready],
			"components": results,
		})
	})
}
