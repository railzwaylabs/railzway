CREATE TABLE integration_catalog (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL, -- 'notification', 'accounting', 'payment', 'crm'
    name TEXT NOT NULL,
    description TEXT,
    logo_url TEXT,
    auth_type TEXT NOT NULL, -- 'oauth2', 'api_key', 'basic'
    schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE integration_connections (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    integration_id TEXT NOT NULL REFERENCES integration_catalog(id),
    name TEXT, -- User-defined name, e.g. 'My Slack #billing'
    config JSONB NOT NULL DEFAULT '{}'::jsonb, -- Non-sensitive config
    encrypted_creds BYTEA, -- Sensitive credentials (API Keys)
    status TEXT NOT NULL DEFAULT 'active', -- 'active', 'error', 'disconnected'
    error_message TEXT,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_integration_connections_org_id ON integration_connections(org_id);
CREATE INDEX idx_integration_connections_integration_id ON integration_connections(integration_id);
CREATE UNIQUE INDEX idx_integration_connections_org_integration ON integration_connections(org_id, integration_id) WHERE status != 'disconnected';
