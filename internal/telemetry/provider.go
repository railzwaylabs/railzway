package telemetry

import (
	"context"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

const (
	otelSDKDisabledEnv            = "OTEL_SDK_DISABLED"
	otelServiceNameEnv            = "OTEL_SERVICE_NAME"
	otelOTLPEndpointEnv           = "OTEL_EXPORTER_OTLP_ENDPOINT"
	otelOTLPTracesEndpointEnv     = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	otelTracesSamplerEnv          = "OTEL_TRACES_SAMPLER"
	otelTracesSamplerArgEnv       = "OTEL_TRACES_SAMPLER_ARG"
	defaultTracingShutdownTimeout = 5 * time.Second
)

func InitTracing(ctx context.Context, serviceName string, logger *zap.Logger) (func(context.Context) error, bool, error) {
	if tracingDisabled() {
		return noopTraceShutdown, false, nil
	}

	if strings.TrimSpace(lookupEnv(otelOTLPTracesEndpointEnv, otelOTLPEndpointEnv)) == "" {
		return noopTraceShutdown, false, nil
	}

	if strings.TrimSpace(os.Getenv(otelServiceNameEnv)) != "" {
		serviceName = strings.TrimSpace(os.Getenv(otelServiceNameEnv))
	}

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(lookupEnv(otelOTLPTracesEndpointEnv, otelOTLPEndpointEnv)))
	if err != nil {
		return nil, false, err
	}

	resource, err := sdkresource.New(
		ctx,
		sdkresource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.namespace", tracerName),
		),
	)
	if err != nil {
		return nil, false, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if logger != nil {
		logger.Info(
			"opentelemetry tracing enabled",
			zap.String("service", serviceName),
			zap.String("endpoint", lookupEnv(otelOTLPTracesEndpointEnv, otelOTLPEndpointEnv)),
			zap.String("sampler", strings.TrimSpace(os.Getenv(otelTracesSamplerEnv))),
			zap.String("sampler_arg", strings.TrimSpace(os.Getenv(otelTracesSamplerArgEnv))),
		)
	}

	return func(stopCtx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(stopCtx, defaultTracingShutdownTimeout)
		defer cancel()
		return provider.Shutdown(shutdownCtx)
	}, true, nil
}

func tracingDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(otelSDKDisabledEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func lookupEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func noopTraceShutdown(context.Context) error {
	return nil
}
