DROP INDEX IF EXISTS idx_customers_test_clock_id;

ALTER TABLE customers
    DROP COLUMN IF EXISTS test_clock_id;

DROP INDEX IF EXISTS idx_test_clocks_org_id;

CREATE UNIQUE INDEX IF NOT EXISTS ux_test_clocks_org_id ON test_clocks(org_id);

ALTER TABLE test_clocks
    DROP COLUMN IF EXISTS name;
