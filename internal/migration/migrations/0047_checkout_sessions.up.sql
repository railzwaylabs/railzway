CREATE TABLE IF NOT EXISTS checkout_sessions (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    customer_id BIGINT,
    provider VARCHAR(50),
    status VARCHAR(20),
    payment_status VARCHAR(20),
    line_items JSONB,
    amount_total BIGINT,
    currency VARCHAR(3),
    success_url TEXT,
    cancel_url TEXT,
    payment_intent_id VARCHAR(255),
    provider_session_id VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_checkout_sessions_provider_id ON checkout_sessions(provider, provider_session_id);
CREATE INDEX IF NOT EXISTS idx_checkout_sessions_org_id ON checkout_sessions(org_id);
