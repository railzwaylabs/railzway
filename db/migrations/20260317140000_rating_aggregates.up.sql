CREATE TABLE IF NOT EXISTS rating_results (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    usage_event_id UUID NOT NULL REFERENCES usage_events(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE RESTRICT,
    plan_price_id UUID NOT NULL REFERENCES plan_prices(id) ON DELETE RESTRICT,
    plan_amount_id UUID REFERENCES plan_amounts(id) ON DELETE RESTRICT,
    meter_id UUID NOT NULL REFERENCES meters(id) ON DELETE RESTRICT,
    currency TEXT NOT NULL,
    quantity NUMERIC NOT NULL CHECK (quantity >= 0),
    unit_amount_cents BIGINT NOT NULL CHECK (unit_amount_cents >= 0),
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    source TEXT NOT NULL DEFAULT 'usage',
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE rating_results
    ADD CONSTRAINT rating_results_window_range
        CHECK (window_end >= window_start);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_rating_results_usage_event
    ON rating_results(usage_event_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_org_id ON rating_results(org_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_customer_id ON rating_results(customer_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_subscription_id ON rating_results(subscription_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_plan_price_id ON rating_results(plan_price_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_plan_amount_id ON rating_results(plan_amount_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_meter_id ON rating_results(meter_id);
CREATE INDEX IF NOT EXISTS idx_rating_results_org_window
    ON rating_results(org_id, window_start, window_end);

CREATE TABLE IF NOT EXISTS usage_aggregates (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE RESTRICT,
    plan_price_id UUID NOT NULL REFERENCES plan_prices(id) ON DELETE RESTRICT,
    plan_amount_id UUID REFERENCES plan_amounts(id) ON DELETE RESTRICT,
    meter_id UUID NOT NULL REFERENCES meters(id) ON DELETE RESTRICT,
    currency TEXT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    quantity NUMERIC NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
    last_event_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE usage_aggregates
    ADD CONSTRAINT usage_aggregates_period_range
        CHECK (period_end >= period_start);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_usage_aggregates_unique
    ON usage_aggregates(org_id, customer_id, plan_price_id, plan_amount_id, meter_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_usage_aggregates_org_id ON usage_aggregates(org_id);
CREATE INDEX IF NOT EXISTS idx_usage_aggregates_customer_id ON usage_aggregates(customer_id);
CREATE INDEX IF NOT EXISTS idx_usage_aggregates_subscription_id ON usage_aggregates(subscription_id);
CREATE INDEX IF NOT EXISTS idx_usage_aggregates_period
    ON usage_aggregates(org_id, period_start, period_end);
