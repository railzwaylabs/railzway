package ctxlogger

import (
	"context"
	"sync/atomic"

	"github.com/smallbiznis/railzway/pkg/telemetry/correlation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type eventSubjectKey struct{}

var serviceName atomic.Pointer[string]

// SetServiceName configures the service name added to every log entry.
func SetServiceName(name string) {
	serviceName.Store(&name)
}

// ContextWithEventSubject annotates the context with the current event subject.
func ContextWithEventSubject(ctx context.Context, subject string) context.Context {
	if subject == "" {
		return ctx
	}
	return context.WithValue(ctx, eventSubjectKey{}, subject)
}

// FromContext returns a logger enriched with tracing and correlation metadata from context.
func FromContext(ctx context.Context) *zap.Logger {
	return WithContext(ctx, zap.L())
}

// WithContext enriches the provided logger using metadata in the context.
func WithContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	if ctx == nil {
		return base
	}

	fields := make([]zap.Field, 0, 6)
	fields = append(fields, ExtractCorrelation(ctx))
	fields = append(fields, ExtractTrace(ctx)...)

	name := "unknown"
	if namePtr := serviceName.Load(); namePtr != nil {
		name = *namePtr
	}
	fields = append(fields, zap.String("service", name), zap.String("service_name", name))

	if subject, ok := ctx.Value(eventSubjectKey{}).(string); ok && subject != "" {
		fields = append(fields, zap.String("event_subject", subject))
	}

	return base.With(fields...)
}

// ExtractCorrelation pulls the correlation ID from the context.
func ExtractCorrelation(ctx context.Context) zap.Field {
	cid := correlation.ExtractCorrelationID(ctx)
	if cid == "" {
		_, cid = correlation.EnsureCorrelationID(ctx)
	}
	return zap.String("correlation_id", cid)
}

// ExtractTrace pulls tracing identifiers from the context span.
func ExtractTrace(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return []zap.Field{zap.String("trace_id", ""), zap.String("span_id", "")}
	}

	return []zap.Field{
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	}
}
