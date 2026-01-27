-- Add test_clock_id to billing_cycles
ALTER TABLE billing_cycles ADD COLUMN test_clock_id BIGINT;
CREATE INDEX idx_billing_cycles_test_clock_id ON billing_cycles(test_clock_id);
