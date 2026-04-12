ALTER TABLE apps_installations
ADD COLUMN IF NOT EXISTS auth_method TEXT NOT NULL DEFAULT 'api_keys';

CREATE INDEX IF NOT EXISTS idx_apps_installations_auth_method
    ON apps_installations(auth_method);
