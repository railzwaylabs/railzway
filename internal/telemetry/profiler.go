package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"github.com/grafana/pyroscope-go"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// StartProfiler initializes the profiler based on the environment.
// In Staging/Production, it uses Grafana Pyroscope SDK (Push Mode).
// In Local/Development, it exposes standard pprof on the given port.
func StartProfiler(port int) interface{} {
	return func(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) {
		if cfg.AppEnv.IsProduction() {
			startPyroscope(lc, cfg, logger)
		} else {
			startLocalPprof(lc, logger, port)
		}
	}
}

func startPyroscope(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) {
	serviceName := "railzway"
	if cfg != nil && strings.TrimSpace(cfg.AppName) != "" {
		serviceName = strings.TrimSpace(cfg.AppName)
	}

	pyroscopeAddress := os.Getenv("PYROSCOPE_SERVER_ADDRESS")
	if pyroscopeAddress == "" {
		logger.Warn("pyroscope address not set, continuous profiler will not start (set PYROSCOPE_SERVER_ADDRESS)")
		return
	}

	var prof *pyroscope.Profiler

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p, err := pyroscope.Start(pyroscope.Config{
				ApplicationName: serviceName,
				ServerAddress:   pyroscopeAddress,
				Logger:          nil,
				ProfileTypes: []pyroscope.ProfileType{
					pyroscope.ProfileCPU,
					pyroscope.ProfileAllocObjects,
					pyroscope.ProfileAllocSpace,
					pyroscope.ProfileInuseObjects,
					pyroscope.ProfileInuseSpace,
				},
			})
			if err != nil {
				logger.Error("failed to start pyroscope", zap.Error(err))
				return nil
			}
			prof = p
			logger.Info("pyroscope continuous profiler started", zap.String("address", pyroscopeAddress))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if prof != nil {
				_ = prof.Stop()
				logger.Info("pyroscope continuous profiler stopped")
			}
			return nil
		},
	})
}

func startLocalPprof(lc fx.Lifecycle, logger *zap.Logger, port int) {
	// For security, bind specifically to localhost so it is not publicly accessible.
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info("local profiler (pprof) server started", zap.String("addr", addr))
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("profiler server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("local profiler server stopping", zap.String("addr", addr))
			return server.Shutdown(ctx)
		},
	})
}
