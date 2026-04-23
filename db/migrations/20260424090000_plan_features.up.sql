CREATE TABLE IF NOT EXISTS plan_features (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    limit_numeric DOUBLE PRECISION,
    limit_unit TEXT,
    reset_period TEXT NOT NULL DEFAULT 'none',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT plan_features_reset_period_check CHECK (reset_period IN ('none', 'day', 'month', 'billing_period'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_plan_features_org_plan_feature ON plan_features(org_id, plan_id, feature_id);
CREATE INDEX IF NOT EXISTS idx_plan_features_org_plan ON plan_features(org_id, plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_features_org_feature ON plan_features(org_id, feature_id);
