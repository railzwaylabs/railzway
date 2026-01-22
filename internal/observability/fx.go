package observability

import (
	"github.com/smallbiznis/railzway/internal/observability/logger"
	"github.com/smallbiznis/railzway/internal/observability/metrics"
	"github.com/smallbiznis/railzway/internal/observability/tracing"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"
)

var Module = fx.Module("observability",
	fx.Provide(
		LoadConfig,
		provideLoggerConfig,
		logger.New,
		provideTracingConfig,
		tracing.NewProvider,
		provideMetricsConfig,
		metrics.NewProvider,
		metrics.New,
		metrics.NewHTTPMetrics,
	),
	fx.Invoke(ensureTracingProvider),
	fx.Invoke(ensureSchedulerMetrics),
)

func ensureTracingProvider(_ *sdktrace.TracerProvider) {}

func provideLoggerConfig(cfg Config) logger.Config {
	return logger.Config{
		ServiceName:         cfg.ServiceName,
		Environment:         cfg.Environment,
		Version:             cfg.Version,
		Level:               cfg.LogLevel,
		Format:              cfg.LogFormat,
		Debug:               cfg.Debug(),
		IncludeCaller:       true,
		IncludeStackOnError: cfg.Debug(),
	}
}

func provideTracingConfig(cfg Config) tracing.Config {
	return tracing.Config{
		Enabled:          cfg.OtelEnabled,
		ServiceName:      cfg.ServiceName,
		ServiceVersion:   cfg.Version,
		Environment:      cfg.Environment,
		ExporterEndpoint: cfg.OtelExporterEndpoint,
		ExporterProtocol: cfg.OtelExporterProtocol,
		SamplingRatio:    cfg.OtelSamplingRatio,
	}
}

func provideMetricsConfig(cfg Config) metrics.Config {
	return metrics.Config{
		Enabled:          cfg.OtelEnabled,
		ExporterEndpoint: cfg.OtelExporterEndpoint,
		ExporterProtocol: cfg.OtelExporterProtocol,
		ServiceName:      cfg.ServiceName,
		Environment:      cfg.Environment,
	}
}

func ensureSchedulerMetrics(cfg metrics.Config) {
	metrics.SchedulerWithConfig(cfg)
}
