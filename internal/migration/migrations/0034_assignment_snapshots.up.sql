-- Add snapshot metadata to assignments for task stability
-- This ensures claimed tasks display consistent data even when billing state changes

ALTER TABLE billing_operation_assignments
ADD COLUMN snapshot_metadata JSONB;

COMMENT ON COLUMN billing_operation_assignments.snapshot_metadata IS 
'Snapshot of entity state at claim time (amount_due, days_overdue, invoice_number, customer_name) - ensures task stability and prevents claimed work from disappearing when billing data changes';

-- Index for querying assignments by status (for My Work and Recently Resolved views)
CREATE INDEX IF NOT EXISTS idx_billing_assignments_user_status 
ON billing_operation_assignments(org_id, assigned_to, status);

-- Note: Index for Recently Resolved view (idx_billing_assignments_resolved_at) 
-- will be created in migration 0035 after resolved_at column is added
