CREATE TABLE IF NOT EXISTS org_bootstrap_state (
  org_id BIGINT PRIMARY KEY,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  activated_at TIMESTAMPTZ,
  suspended_at TIMESTAMPTZ,
  terminated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_org_bootstrap_state_status ON org_bootstrap_state(status);

INSERT INTO org_bootstrap_state (org_id, status, created_at, activated_at)
SELECT id, 'active', created_at, created_at
FROM organizations
ON CONFLICT (org_id) DO NOTHING;
