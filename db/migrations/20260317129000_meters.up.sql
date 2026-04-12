CREATE TABLE IF NOT EXISTS meters (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    aggregation TEXT NOT NULL,
    unit TEXT NOT NULL,
    idempotency_key TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN meters.code IS 'Unique meter identifier within org';
COMMENT ON COLUMN meters.aggregation IS 'How to aggregate: sum|last|max|min';
COMMENT ON COLUMN meters.unit IS 'Unit of measure (e.g., "gb", "requests", "users")';
COMMENT ON COLUMN meters.idempotency_key IS 'Client-provided dedupe key for meter creation';

CREATE UNIQUE INDEX IF NOT EXISTS ux_meters_org_code ON meters(org_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_meters_idempotency_key ON meters(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_meters_org_id ON meters(org_id);
