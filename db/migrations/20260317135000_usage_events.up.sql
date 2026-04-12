CREATE TABLE IF NOT EXISTS usage_events (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    meter_id UUID NOT NULL REFERENCES meters(id) ON DELETE RESTRICT,
    meter_code TEXT NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'accepted' CHECK (status IN ('accepted', 'enriched', 'rated')),
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN usage_events.meter_code IS 'Snapshot of meter code at ingest time for audit/debug';
COMMENT ON COLUMN usage_events.meter_id IS 'FK to meters (source of truth)';
COMMENT ON COLUMN usage_events.customer_id IS 'Customer that generated the usage event';
COMMENT ON COLUMN usage_events.value IS 'Usage quantity/value reported by client';
COMMENT ON COLUMN usage_events.recorded_at IS 'When usage occurred (client time)';
COMMENT ON COLUMN usage_events.status IS 'Processing state: accepted|enriched|rated';
COMMENT ON COLUMN usage_events.idempotency_key IS 'Client-provided dedupe key';

CREATE UNIQUE INDEX IF NOT EXISTS uidx_usage_events_idempotency_key
    ON usage_events (org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_usage_events_org_id ON usage_events(org_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_meter_id ON usage_events(meter_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_customer_id ON usage_events(customer_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_org_meter_recorded
    ON usage_events(org_id, meter_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_events_org_customer_recorded
    ON usage_events(org_id, customer_id, recorded_at DESC);
