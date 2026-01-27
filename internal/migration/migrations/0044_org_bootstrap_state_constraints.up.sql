DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_org_bootstrap_state_status'
  ) THEN
    ALTER TABLE org_bootstrap_state
      ADD CONSTRAINT chk_org_bootstrap_state_status
      CHECK (status IN ('initializing', 'active', 'suspended', 'terminated'));
  END IF;
END $$;
