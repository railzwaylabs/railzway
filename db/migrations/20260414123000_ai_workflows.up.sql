CREATE TABLE IF NOT EXISTS ai_workflows (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    summary TEXT,
    intent TEXT NOT NULL,
    status TEXT NOT NULL,
    source_run_id UUID REFERENCES ai_assistant_runs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_workflows_org_created ON ai_workflows(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_workflows_status ON ai_workflows(status);

CREATE TABLE IF NOT EXISTS ai_workflow_actions (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES ai_workflows(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    label TEXT NOT NULL,
    status TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    "order" INT NOT NULL,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_workflow_actions_workflow ON ai_workflow_actions(workflow_id);

CREATE TABLE IF NOT EXISTS ai_workflow_approvals (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES ai_workflows(id) ON DELETE CASCADE,
    actor_id TEXT NOT NULL,
    status TEXT NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_workflow_approvals_workflow ON ai_workflow_approvals(workflow_id);
