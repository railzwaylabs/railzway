CREATE TABLE IF NOT EXISTS subscription_periods (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE subscription_periods
    ADD CONSTRAINT subscription_periods_range
        CHECK (period_end >= period_start);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_subscription_periods_unique
    ON subscription_periods(org_id, subscription_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_subscription_periods_org_id ON subscription_periods(org_id);
CREATE INDEX IF NOT EXISTS idx_subscription_periods_subscription_id ON subscription_periods(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_periods_range
    ON subscription_periods(org_id, subscription_id, period_start, period_end);
