CREATE TABLE IF NOT EXISTS test_clock_state (
    test_clock_id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    simulated_time TIMESTAMPTZ NOT NULL,
    advancing_to TIMESTAMPTZ,
    status TEXT NOT NULL,
    last_error TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_test_clock_state_status CHECK (status IN ('idle', 'advancing', 'succeeded', 'failed'))
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_test_clock_state_status'
  ) THEN
    ALTER TABLE test_clock_state
      ADD CONSTRAINT chk_test_clock_state_status
      CHECK (status IN ('idle', 'advancing', 'succeeded', 'failed'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_test_clock_state_org_status
    ON test_clock_state(org_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS ux_test_clock_state_org_advancing
    ON test_clock_state(org_id)
    WHERE status = 'advancing';

INSERT INTO test_clock_state (test_clock_id, org_id, simulated_time, status, updated_at)
SELECT tc.id, tc.org_id, tc.simulated_time, 'idle', NOW()
FROM test_clocks tc
WHERE NOT EXISTS (
    SELECT 1
    FROM test_clock_state tcs
    WHERE tcs.test_clock_id = tc.id
);
