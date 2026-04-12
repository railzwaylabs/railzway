package orgcontext

import (
	"context"

	"github.com/google/uuid"
)

type key string

const orgIDKey key = "org_id"

// WithOrgID stores orgID in context.
func WithOrgID(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, orgIDKey, orgID)
}

// OrgIDFromContext reads orgID from context.
func OrgIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(orgIDKey).(uuid.UUID)
	return id, ok
}
