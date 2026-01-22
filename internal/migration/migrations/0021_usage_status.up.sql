ALTER TABLE usage_events
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'accepted',
    ADD COLUMN IF NOT EXISTS error TEXT;

DROP INDEX IF EXISTS uidx_usage_idempotency_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_events_idempotency
    ON usage_events (org_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_usage_events_rating_lookup ON usage_events(org_id, subscription_id, meter_id, recorded_at, status) INCLUDE (value);
CREATE INDEX IF NOT EXISTS idx_usage_events_org_meter_recorded ON usage_events(org_id, meter_id, recorded_at DESC);
