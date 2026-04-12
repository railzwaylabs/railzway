package auditlog

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	actorTypeKey contextKey = "audit_actor_type"
	actorIDKey   contextKey = "audit_actor_id"
	requestIDKey contextKey = "audit_request_id"
	reasonKey    contextKey = "audit_reason"
	metadataKey  contextKey = "audit_metadata"
)

func WithActor(ctx context.Context, actorType string, actorID *uuid.UUID) context.Context {
	if actorType != "" {
		ctx = context.WithValue(ctx, actorTypeKey, actorType)
	}
	if actorID != nil && *actorID != uuid.Nil {
		ctx = context.WithValue(ctx, actorIDKey, *actorID)
	}
	return ctx
}

func ActorFromContext(ctx context.Context) (string, *uuid.UUID) {
	var actorType string
	if raw := ctx.Value(actorTypeKey); raw != nil {
		if val, ok := raw.(string); ok {
			actorType = val
		}
	}

	var actorID *uuid.UUID
	if raw := ctx.Value(actorIDKey); raw != nil {
		switch v := raw.(type) {
		case uuid.UUID:
			if v != uuid.Nil {
				actorID = &v
			}
		case *uuid.UUID:
			if v != nil && *v != uuid.Nil {
				actorID = v
			}
		}
	}
	return actorType, actorID
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if raw := ctx.Value(requestIDKey); raw != nil {
		if val, ok := raw.(string); ok {
			return val
		}
	}
	return ""
}

func WithReason(ctx context.Context, reason string) context.Context {
	if reason == "" {
		return ctx
	}
	return context.WithValue(ctx, reasonKey, reason)
}

func ReasonFromContext(ctx context.Context) string {
	if raw := ctx.Value(reasonKey); raw != nil {
		if val, ok := raw.(string); ok {
			return val
		}
	}
	return ""
}

func WithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	if len(metadata) == 0 {
		return ctx
	}
	return context.WithValue(ctx, metadataKey, metadata)
}

func MetadataFromContext(ctx context.Context) map[string]interface{} {
	if raw := ctx.Value(metadataKey); raw != nil {
		if val, ok := raw.(map[string]interface{}); ok {
			return val
		}
	}
	return nil
}
