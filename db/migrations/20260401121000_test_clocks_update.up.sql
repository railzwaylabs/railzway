DROP INDEX IF EXISTS ux_test_clocks_org_id;

ALTER TABLE test_clocks
    ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT 'Test clock';

CREATE INDEX IF NOT EXISTS idx_test_clocks_org_id ON test_clocks(org_id);

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS test_clock_id UUID NULL REFERENCES test_clocks(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_customers_test_clock_id ON customers(test_clock_id);
