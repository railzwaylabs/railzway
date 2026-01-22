-- Add resolved_at and resolved_by columns for Recently Resolved view
-- These columns track when and by whom an assignment was completed (resolved, released, or escalated)

ALTER TABLE billing_operation_assignments
  ADD COLUMN resolved_at TIMESTAMPTZ,
  ADD COLUMN resolved_by TEXT;

-- Migrate existing data: copy released_at to resolved_at for terminal states
UPDATE billing_operation_assignments
SET resolved_at = released_at,
    resolved_by = released_by
WHERE status IN ('resolved', 'released', 'escalated')
  AND released_at IS NOT NULL;

-- The index already exists from migration 0034, but it was broken (referenced non-existent column)
-- Drop and recreate to ensure it's valid
DROP INDEX IF EXISTS idx_billing_assignments_resolved_at;
CREATE INDEX idx_billing_assignments_resolved_at 
  ON billing_operation_assignments(org_id, assigned_to, resolved_at DESC) 
  WHERE status IN ('resolved', 'released', 'escalated');
