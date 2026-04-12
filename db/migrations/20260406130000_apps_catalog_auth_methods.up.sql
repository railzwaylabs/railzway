ALTER TABLE apps_catalog
ADD COLUMN IF NOT EXISTS auth_methods JSONB NOT NULL DEFAULT '["api_keys"]';

UPDATE apps_catalog
SET auth_methods = '["oauth2","api_keys"]'
WHERE id = 'payment.stripe';

UPDATE apps_catalog
SET auth_methods = '["basic_auth"]'
WHERE id = 'email.smtp';

UPDATE apps_catalog
SET auth_methods = '["api_keys"]'
WHERE auth_methods IS NULL;
