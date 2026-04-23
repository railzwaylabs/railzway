CREATE TABLE IF NOT EXISTS billing_segments (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'customer',
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT billing_segments_scope_check CHECK (scope IN ('any', 'customer', 'subscription'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_billing_segments_org_key ON billing_segments(org_id, key);
CREATE INDEX IF NOT EXISTS idx_billing_segments_org_active ON billing_segments(org_id, active);
CREATE INDEX IF NOT EXISTS idx_billing_segments_org_scope ON billing_segments(org_id, scope);
