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
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/db"
	"github.com/railzwaylabs/railzway/internal/httpmiddleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.RegisterFor("checkout"),
			newLogger,
			fx.Annotate(newRouter, fx.ParamTags("", "", "")),
		),
		db.Module,
		fx.Invoke(registerLoggerLifecycle),
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

func newRouter(cfg *config.Config, logger *zap.Logger, dbConn *gorm.DB) *gin.Engine {
	if cfg.AppEnv.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpmiddleware.SecurityHeadersWithCSP(cfg, cfg.CSPExtraDirectives, httpmiddleware.CSPProfileCheckout))
	router.Use(httpmiddleware.ZapRequestLogger(logger))
	router.Use(httpmiddleware.RequireTLS(cfg))

	registerHealthRoutes(router, dbConn)

	distDir := filepath.Join("apps", "checkout", "dist")
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
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
		logger.Info("checkout ui static serving enabled", zap.String("dir", distDir))
	} else {
		logger.Warn("checkout ui dist not found, skipping static serve", zap.String("dir", distDir))
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

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("checkout server started", zap.String("addr", addr), zap.String("tls_mode", string(cfg.AppTLSMode)))
				var err error
				if cfg.AppTLSMode.IsDirect() {
					err = server.ListenAndServeTLS(cfg.AppTLSCertFile, cfg.AppTLSKeyFile)
				} else {
					err = server.ListenAndServe()
				}
				if err != nil && err != http.ErrServerClosed {
					logger.Error("checkout server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("checkout server stopping", zap.String("addr", addr))
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
