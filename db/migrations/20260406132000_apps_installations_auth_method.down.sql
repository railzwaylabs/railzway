DROP INDEX IF EXISTS idx_apps_installations_auth_method;
ALTER TABLE apps_installations
DROP COLUMN IF EXISTS auth_method;
