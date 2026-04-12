CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    rollout INTEGER NOT NULL DEFAULT 0 CHECK (rollout >= 0 AND rollout <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_feature_flags_org_key
    ON feature_flags(org_id, key);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_feature_flags_global_key
    ON feature_flags(key) WHERE org_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_feature_flags_org_id ON feature_flags(org_id);

CREATE TABLE IF NOT EXISTS feature_flag_audits (
    id UUID PRIMARY KEY,
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    enabled BOOLEAN NOT NULL,
    rollout INTEGER NOT NULL CHECK (rollout >= 0 AND rollout <= 100),
    actor_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feature_flag_audits_org_id ON feature_flag_audits(org_id);
CREATE INDEX IF NOT EXISTS idx_feature_flag_audits_key ON feature_flag_audits(key);
