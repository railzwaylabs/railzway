CREATE TABLE IF NOT EXISTS system_bootstrap_state (
  id BOOLEAN PRIMARY KEY DEFAULT TRUE,
  status TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  checksum TEXT,
  activated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS currencies (
  code CHAR(3) PRIMARY KEY,
  name TEXT NOT NULL,
  symbol TEXT,
  minor_unit SMALLINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS billing_cycle_types (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ledger_account_types (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
