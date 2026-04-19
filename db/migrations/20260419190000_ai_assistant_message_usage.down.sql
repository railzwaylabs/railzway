ALTER TABLE ai_assistant_messages
    DROP COLUMN IF EXISTS latency_ms,
    DROP COLUMN IF EXISTS usage;

