package log

import (
	"context"

	"github.com/smallbiznis/railzway/internal/config"
	"github.com/smallbiznis/railzway/pkg/log/ctxlogger"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Module provides a zap logger configured for production.
var Module = fx.Provide(NewLogger)

// NewLogger returns a production zap logger with consistent JSON output and replaces globals.
func NewLogger(cfg config.Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()
	zapCfg.Encoding = "json"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	logger, err := zapCfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, err
	}

	ctxlogger.SetServiceName(cfg.AppName)
	zap.ReplaceGlobals(logger)
	return logger, nil
}

// L returns a context-aware logger with correlation and tracing metadata.
func L(ctx context.Context) *zap.Logger {
	return ctxlogger.FromContext(ctx)
}
