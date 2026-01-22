CREATE TABLE IF NOT EXISTS finops_performance_snapshots (
    id BIGINT PRIMARY KEY,

    -- multi-tenant isolation (logical, not enforced by FK)
    org_id BIGINT NOT NULL,
    user_id TEXT NOT NULL,

    -- period identity
    period_type TEXT NOT NULL CHECK (period_type IN ('daily', 'weekly', 'monthly')),
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,

    -- raw metrics (facts, not interpretation)
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- derived scores per dimension
    scores JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- final aggregated score (0–100)
    total_score INTEGER NOT NULL DEFAULT 0,

    -- audit (immutable snapshot)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Snapshot identity (IMMUTABLE per user & period)
CREATE UNIQUE INDEX ux_finops_snapshots_identity
ON finops_performance_snapshots (
    org_id,
    user_id,
    period_type,
    period_start
);

-- Fast org-level dashboard queries
CREATE INDEX idx_finops_snapshots_org_period
ON finops_performance_snapshots (
    org_id,
    period_type,
    period_start DESC
);

-- Fast user-level performance view
CREATE INDEX idx_finops_snapshots_user_period
ON finops_performance_snapshots (
    org_id,
    user_id,
    period_type,
    period_start DESC
);
