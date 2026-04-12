CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    external_id TEXT,
    name TEXT,
    email TEXT,
    currency TEXT,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_customers_idempotency_key
    ON customers(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_customers_external_id
    ON customers(org_id, external_id) WHERE external_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_customers_email
    ON customers(org_id, email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_customers_org_id ON customers(org_id);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'canceled', 'paused')),
    currency TEXT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    trial_end TIMESTAMPTZ,
    cancel_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_period_range
        CHECK (current_period_end >= current_period_start);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_subscriptions_idempotency_key
    ON subscriptions(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_id ON subscriptions(org_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(org_id, status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end ON subscriptions(org_id, current_period_end);

CREATE TABLE IF NOT EXISTS subscription_items (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    plan_price_id UUID NOT NULL REFERENCES plan_prices(id) ON DELETE RESTRICT,
    quantity NUMERIC NOT NULL DEFAULT 1 CHECK (quantity >= 0),
    start_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_at TIMESTAMPTZ,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE subscription_items
    ADD CONSTRAINT subscription_items_period_range
        CHECK (end_at IS NULL OR end_at >= start_at);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_subscription_items_idempotency_key
    ON subscription_items(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscription_items_org_id ON subscription_items(org_id);
CREATE INDEX IF NOT EXISTS idx_subscription_items_subscription_id ON subscription_items(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_items_plan_price_id ON subscription_items(plan_price_id);
