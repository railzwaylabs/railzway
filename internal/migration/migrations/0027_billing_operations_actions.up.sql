CREATE TABLE IF NOT EXISTS billing_operation_actions (
  id BIGINT PRIMARY KEY,
  org_id BIGINT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id BIGINT NOT NULL,
  action_type TEXT NOT NULL,
  action_bucket DATE NOT NULL,
  idempotency_key TEXT,
  metadata JSONB NOT NULL DEFAULT '{}',
  actor_type TEXT,
  actor_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_billing_operation_actions_bucket
  ON billing_operation_actions(org_id, entity_type, entity_id, action_type, action_bucket);

CREATE UNIQUE INDEX IF NOT EXISTS ux_billing_operation_actions_idempotency
  ON billing_operation_actions(org_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_billing_operation_actions_org_id
  ON billing_operation_actions(org_id);

CREATE INDEX IF NOT EXISTS idx_billing_operation_actions_entity
  ON billing_operation_actions(org_id, entity_type, entity_id);

CREATE TABLE IF NOT EXISTS billing_operation_assignments (
  id BIGINT PRIMARY KEY,
  org_id BIGINT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id BIGINT NOT NULL,
  assigned_to TEXT NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL,
  assignment_expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_billing_operation_assignments_entity
  ON billing_operation_assignments(org_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_billing_operation_assignments_org_id
  ON billing_operation_assignments(org_id);

CREATE INDEX IF NOT EXISTS idx_billing_operation_assignments_assigned_to
  ON billing_operation_assignments(org_id, assigned_to);
