ALTER TABLE apps_catalog
ADD COLUMN IF NOT EXISTS credentials_schema JSONB NOT NULL DEFAULT '{}';

UPDATE apps_catalog
SET credentials_schema = '{
  "api_keys": ["publishable_key","secret_key","webhook_secret"]
}'
WHERE id = 'payment.stripe';

UPDATE apps_catalog
SET credentials_schema = '{
  "api_keys": ["api_key"]
}'
WHERE id = 'email.sendgrid';

UPDATE apps_catalog
SET credentials_schema = '{
  "basic_auth": ["username","password","host","port"]
}'
WHERE id = 'email.smtp';
