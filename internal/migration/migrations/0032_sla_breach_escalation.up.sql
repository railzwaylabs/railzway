ALTER TABLE billing_operation_assignments
  ADD COLUMN breached_at TIMESTAMPTZ,
  ADD COLUMN breach_level TEXT;

CREATE INDEX IF NOT EXISTS idx_billing_operation_assignments_breached_at
  ON billing_operation_assignments(org_id, breached_at);
