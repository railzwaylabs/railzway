CREATE TABLE IF NOT EXISTS ai_assistant_runs (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_ref TEXT,
    time_range TEXT NOT NULL,
    intent TEXT NOT NULL,
    prompt TEXT NOT NULL,
    status TEXT NOT NULL,
    output JSONB NOT NULL DEFAULT '{}',
    error_code TEXT,
    error_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_assistant_runs_org_created ON ai_assistant_runs(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_assistant_runs_status ON ai_assistant_runs(status);
