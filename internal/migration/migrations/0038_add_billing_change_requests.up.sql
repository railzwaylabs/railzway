-- Add billing_change_requests table for approval workflow
CREATE TABLE IF NOT EXISTS billing_change_requests (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    billing_cycle_id BIGINT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    requested_by BIGINT NOT NULL,
    requested_by_name TEXT,
    approved_by BIGINT,
    approved_by_name TEXT,
    reason TEXT NOT NULL,
    rejection_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP,
    executed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_billing_change_requests_org_id ON billing_change_requests(org_id);
CREATE INDEX IF NOT EXISTS idx_billing_change_requests_billing_cycle_id ON billing_change_requests(billing_cycle_id);
CREATE INDEX IF NOT EXISTS idx_billing_change_requests_approved_by ON billing_change_requests(approved_by);
