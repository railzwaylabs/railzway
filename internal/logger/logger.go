package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a structured zap.Logger using the provided level (info, warn, debug, error).
func New(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if level == "" {
		level = "info"
	}

	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)
	return logger, nil
}
