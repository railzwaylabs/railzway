CREATE TABLE IF NOT EXISTS admin_summaries (
    org_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    dashboard JSONB NOT NULL DEFAULT '{}',
    customers JSONB NOT NULL DEFAULT '{}',
    plans JSONB NOT NULL DEFAULT '{}',
    subscriptions JSONB NOT NULL DEFAULT '{}',
    usage JSONB NOT NULL DEFAULT '{}',
    rating JSONB NOT NULL DEFAULT '{}',
    invoices JSONB NOT NULL DEFAULT '{}',
    payments JSONB NOT NULL DEFAULT '{}',
    taxes JSONB NOT NULL DEFAULT '{}',
    audit_logs JSONB NOT NULL DEFAULT '{}',
    settings JSONB NOT NULL DEFAULT '{}',
    source TEXT NOT NULL DEFAULT 'materialized',
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_summaries_refreshed_at ON admin_summaries(refreshed_at);
