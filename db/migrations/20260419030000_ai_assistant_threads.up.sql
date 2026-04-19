CREATE TABLE IF NOT EXISTS ai_assistant_threads (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_assistant_threads_org_user_updated
    ON ai_assistant_threads(org_id, user_id, updated_at DESC)
    WHERE archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_assistant_threads_org_updated
    ON ai_assistant_threads(org_id, updated_at DESC)
    WHERE archived_at IS NULL;

CREATE TABLE IF NOT EXISTS ai_assistant_messages (
    id UUID PRIMARY KEY,
    thread_id UUID NOT NULL REFERENCES ai_assistant_threads(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    role TEXT NOT NULL,
    prompt TEXT,
    tokens JSONB NOT NULL DEFAULT '[]'::jsonb,
    blocks JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ai_assistant_messages_role CHECK (role IN ('user', 'assistant', 'system'))
);

CREATE INDEX IF NOT EXISTS idx_ai_assistant_messages_thread_created
    ON ai_assistant_messages(thread_id, created_at ASC);

CREATE INDEX IF NOT EXISTS idx_ai_assistant_messages_org_user_created
    ON ai_assistant_messages(org_id, user_id, created_at DESC);
