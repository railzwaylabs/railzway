package orgcontext

import (
	"context"
	"strings"

	"github.com/bwmarrin/snowflake"
)

// OrgContextKey is the request context key for the active organization ID.
type OrgContextKey struct{}

// WithOrgID stores the org ID in the context.
func WithOrgID(ctx context.Context, orgID int64) context.Context {
	return context.WithValue(ctx, OrgContextKey{}, orgID)
}

// OrgIDFromContext returns the org ID from context, if set.
func OrgIDFromContext(ctx context.Context) (snowflake.ID, bool) {
	if ctx == nil {
		return 0, false
	}

	value := ctx.Value(OrgContextKey{})
	if value != nil {
		switch typed := value.(type) {
		case int64:
			return snowflake.ID(typed), true
		case snowflake.ID:
			return typed, true
		case string:
			parsed, err := snowflake.ParseString(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}

	raw := ctx.Value("org_id")
	if raw == nil {
		return 0, false
	}
	switch typed := raw.(type) {
	case int64:
		return snowflake.ID(typed), true
	case snowflake.ID:
		return typed, true
	case string:
		parsed, err := snowflake.ParseString(strings.TrimSpace(typed))
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}
