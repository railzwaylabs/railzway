ALTER TABLE billing_operation_assignments
  ADD COLUMN status TEXT NOT NULL DEFAULT 'assigned',
  ADD COLUMN released_at TIMESTAMPTZ,
  ADD COLUMN released_by TEXT,
  ADD COLUMN release_reason TEXT,
  ADD COLUMN last_action_at TIMESTAMPTZ;

ALTER TABLE billing_operation_actions
  ADD COLUMN assignment_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_billing_operation_assignments_status
  ON billing_operation_assignments(org_id, status);

CREATE INDEX IF NOT EXISTS idx_billing_operation_actions_assignment
  ON billing_operation_actions(org_id, assignment_id);
