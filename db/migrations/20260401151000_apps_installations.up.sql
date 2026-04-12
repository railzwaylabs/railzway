CREATE TABLE IF NOT EXISTS apps_installations (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    app_id TEXT NOT NULL REFERENCES apps_catalog(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    config JSONB NOT NULL DEFAULT '{}',
    credentials JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_apps_installations_org_app
    ON apps_installations(org_id, app_id);
CREATE INDEX IF NOT EXISTS idx_apps_installations_org_id
    ON apps_installations(org_id);
CREATE INDEX IF NOT EXISTS idx_apps_installations_app_id
    ON apps_installations(app_id);
