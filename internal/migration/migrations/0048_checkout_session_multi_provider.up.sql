-- Add new fields to checkout_sessions table for multi-provider support
ALTER TABLE checkout_sessions
ADD COLUMN IF NOT EXISTS subscription_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS client_reference_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS expired_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_checkout_sessions_subscription_id ON checkout_sessions(subscription_id);
CREATE INDEX IF NOT EXISTS idx_checkout_sessions_client_reference_id ON checkout_sessions(client_reference_id);
