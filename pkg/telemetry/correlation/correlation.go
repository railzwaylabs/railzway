package correlation

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
	eventv1 "github.com/smallbiznis/go-genproto/smallbiznis/event/v1"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/structpb"
)

// correlationKey is an unexported type for context keys within this package.
type correlationKey struct{}

// ExtractCorrelationID fetches a correlation ID from the context if present.
func ExtractCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(correlationKey{}).(string); ok {
		return val
	}
	return ""
}

// ContextWithCorrelationID sets the correlation ID onto the context.
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationKey{}, id)
}

// InjectCorrelationID is kept for backwards compatibility and delegates to ContextWithCorrelationID.
func InjectCorrelationID(ctx context.Context, id string) context.Context {
	return ContextWithCorrelationID(ctx, id)
}

// EnsureCorrelationID guarantees a correlation ID on the context, generating one when missing.
func EnsureCorrelationID(ctx context.Context) (context.Context, string) {
	cid := ExtractCorrelationID(ctx)
	if cid == "" {
		cid = ulid.Make().String()
	}
	return ContextWithCorrelationID(ctx, cid), cid
}

// InjectTraceIntoEvent augments the event metadata with correlation and tracing identifiers.
func InjectTraceIntoEvent(evt *eventv1.Event, span trace.Span) {
	if evt == nil {
		return
	}
	if evt.Metadata == nil {
		evt.Metadata = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if evt.Metadata.Fields == nil {
		evt.Metadata.Fields = map[string]*structpb.Value{}
	}

	cid := ""
	if current, ok := evt.Metadata.Fields["correlation_id"]; ok {
		cid = current.GetStringValue()
	}
	if cid == "" {
		cid = ulid.Make().String()
	}

	ctx := span.SpanContext()
	evt.Metadata.Fields["correlation_id"] = structpb.NewStringValue(cid)
	evt.Metadata.Fields["trace_id"] = structpb.NewStringValue(ctx.TraceID().String())
	evt.Metadata.Fields["span_id"] = structpb.NewStringValue(ctx.SpanID().String())
	evt.Metadata.Fields["published_at"] = structpb.NewStringValue(time.Now().UTC().Format(time.RFC3339))
}

// ContextWithRemoteSpan seeds the context with a remote span if valid identifiers are provided.
func ContextWithRemoteSpan(ctx context.Context, traceIDHex, spanIDHex string) context.Context {
	if traceIDHex == "" || spanIDHex == "" {
		return ctx
	}

	traceID, err := trace.TraceIDFromHex(traceIDHex)
	if err != nil {
		return ctx
	}
	spanID, err := trace.SpanIDFromHex(spanIDHex)
	if err != nil {
		return ctx
	}

	parent := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled, Remote: true})
	return trace.ContextWithSpanContext(ctx, parent)
}
