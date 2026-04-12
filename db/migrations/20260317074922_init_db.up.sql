CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    external_id TEXT UNIQUE,
    provider TEXT,
    display_name TEXT,
    email TEXT UNIQUE,
    password_hash TEXT,
    last_password_changed TIMESTAMPTZ,
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    session_token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address TEXT,
    active_org_id UUID,
    org_ids JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_active_org_id ON sessions(active_org_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    country_code TEXT,
    timezone_name TEXT,
    support_email TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_organizations_slug ON organizations(slug);

ALTER TABLE sessions
    ADD CONSTRAINT fk_sessions_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_sessions_active_org_id
        FOREIGN KEY (active_org_id) REFERENCES organizations(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_org_user ON organization_members(org_id, user_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_org_id ON organization_members(org_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_user_id ON organization_members(user_id);

CREATE TABLE IF NOT EXISTS organization_invites (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_organization_invites_org_id ON organization_invites(org_id);
CREATE INDEX IF NOT EXISTS idx_organization_invites_invited_by ON organization_invites(invited_by);
CREATE INDEX IF NOT EXISTS idx_organization_invites_email ON organization_invites(email);

CREATE TABLE IF NOT EXISTS organization_billing_preferences (
    org_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    timezone TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS organization_links (
    id UUID PRIMARY KEY,
    parent_org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    child_org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    mode TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_organization_links_parent_child
    ON organization_links(parent_org_id, child_org_id);
CREATE INDEX IF NOT EXISTS idx_organization_links_parent_org_id
    ON organization_links(parent_org_id);
CREATE INDEX IF NOT EXISTS idx_organization_links_child_org_id
    ON organization_links(child_org_id);
