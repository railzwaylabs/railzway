CREATE TABLE IF NOT EXISTS ai_scheduled_jobs (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    task_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    schedule_cron TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ai_scheduled_jobs_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_ai_scheduled_jobs_org_id ON ai_scheduled_jobs(org_id);
CREATE INDEX IF NOT EXISTS idx_ai_scheduled_jobs_status ON ai_scheduled_jobs(status);
CREATE INDEX IF NOT EXISTS idx_ai_scheduled_jobs_next_run_at ON ai_scheduled_jobs(next_run_at);
