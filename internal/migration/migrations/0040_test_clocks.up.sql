-- Create test_clocks table
CREATE TABLE test_clocks (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    simulated_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_test_clocks_org_id ON test_clocks(org_id);

-- Add test_clock_id to core entities
ALTER TABLE customers ADD COLUMN test_clock_id BIGINT;
CREATE INDEX idx_customers_test_clock_id ON customers(test_clock_id);

ALTER TABLE subscriptions ADD COLUMN test_clock_id BIGINT;
CREATE INDEX idx_subscriptions_test_clock_id ON subscriptions(test_clock_id);

ALTER TABLE subscription_items ADD COLUMN test_clock_id BIGINT;
CREATE INDEX idx_subscription_items_test_clock_id ON subscription_items(test_clock_id);

ALTER TABLE invoices ADD COLUMN test_clock_id BIGINT;
CREATE INDEX idx_invoices_test_clock_id ON invoices(test_clock_id);

ALTER TABLE ledger_entries ADD COLUMN test_clock_id BIGINT;
CREATE INDEX idx_ledger_entries_test_clock_id ON ledger_entries(test_clock_id);
