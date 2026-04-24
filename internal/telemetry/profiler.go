package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"syscall"

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
		if cfg.App.Env.IsProduction() {
			startPyroscope(lc, cfg, logger)
		} else {
			startLocalPprof(lc, logger, port)
		}
	}
}

func startPyroscope(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) {
	serviceName := "railzway"
	if cfg != nil && strings.TrimSpace(cfg.App.Name) != "" {
		serviceName = strings.TrimSpace(cfg.App.Name)
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
	addr, enabled := localPprofAddr(port)
	if !enabled {
		logger.Info("local profiler (pprof) disabled")
		return
	}

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
	started := false

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				if errors.Is(err, syscall.EADDRINUSE) {
					logger.Warn("local profiler (pprof) address already in use; continuing without pprof", zap.String("addr", addr), zap.Error(err))
					return nil
				}
				logger.Warn("local profiler (pprof) unavailable; continuing without pprof", zap.String("addr", addr), zap.Error(err))
				return nil
			}
			started = true
			logger.Info("local profiler (pprof) server started", zap.String("addr", addr))

			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					logger.Error("profiler server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if !started {
				return nil
			}
			logger.Info("local profiler server stopping", zap.String("addr", addr))
			return server.Shutdown(ctx)
		},
	})
}

func localPprofAddr(port int) (string, bool) {
	addr := strings.TrimSpace(os.Getenv("RAILZWAY_PPROF_ADDR"))
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("PPROF_ADDR"))
	}
	switch strings.ToLower(addr) {
	case "off", "false", "disabled", "disable", "none":
		return "", false
	case "":
		return fmt.Sprintf("127.0.0.1:%d", port), true
	default:
		return addr, true
	}
}
