CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_plans_org_code ON plans(org_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_plans_idempotency_key ON plans(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_plans_org_id ON plans(org_id);
CREATE INDEX IF NOT EXISTS idx_plans_product_id ON plans(product_id);

CREATE TABLE IF NOT EXISTS plan_prices (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    meter_id UUID REFERENCES meters(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT,
    description TEXT,
    price_type TEXT NOT NULL CHECK (price_type IN ('flat', 'usage', 'tiered')),
    billing_interval TEXT NOT NULL CHECK (billing_interval IN ('day', 'week', 'month', 'year')),
    billing_interval_count INTEGER NOT NULL DEFAULT 1 CHECK (billing_interval_count >= 1),
    aggregate_usage TEXT,
    billing_unit TEXT,
    meter_code TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_plan_prices_org_code ON plan_prices(org_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_plan_prices_idempotency_key ON plan_prices(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_plan_prices_org_id ON plan_prices(org_id);
CREATE INDEX IF NOT EXISTS idx_plan_prices_plan_id ON plan_prices(plan_id);

CREATE TABLE IF NOT EXISTS plan_amounts (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_price_id UUID NOT NULL REFERENCES plan_prices(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    unit_amount_cents NUMERIC(28,12) NOT NULL CHECK (unit_amount_cents >= 0),
    minimum_amount_cents NUMERIC(28,12) CHECK (minimum_amount_cents IS NULL OR minimum_amount_cents >= 0),
    maximum_amount_cents NUMERIC(28,12) CHECK (maximum_amount_cents IS NULL OR maximum_amount_cents >= 0),
    effective_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    effective_to TIMESTAMPTZ,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE plan_amounts
    ADD CONSTRAINT plan_amounts_min_lte_max
        CHECK (
            minimum_amount_cents IS NULL
            OR maximum_amount_cents IS NULL
            OR minimum_amount_cents <= maximum_amount_cents
        ),
    ADD CONSTRAINT plan_amounts_effective_range
        CHECK (
            effective_to IS NULL
            OR effective_to >= effective_from
        );

CREATE INDEX IF NOT EXISTS idx_plan_amounts_org_id ON plan_amounts(org_id);
CREATE INDEX IF NOT EXISTS idx_plan_amounts_plan_price_id ON plan_amounts(plan_price_id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_plan_amounts_idempotency_key ON plan_amounts(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS plan_tiers (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_price_id UUID NOT NULL REFERENCES plan_prices(id) ON DELETE CASCADE,
    tier_mode TEXT NOT NULL DEFAULT 'graduated' CHECK (tier_mode IN ('graduated', 'volume')),
    start_quantity NUMERIC NOT NULL CHECK (start_quantity >= 0),
    end_quantity NUMERIC CHECK (end_quantity IS NULL OR end_quantity >= start_quantity),
    unit_amount_cents NUMERIC(28,12) CHECK (unit_amount_cents IS NULL OR unit_amount_cents >= 0),
    flat_amount_cents NUMERIC(28,12) CHECK (flat_amount_cents IS NULL OR flat_amount_cents >= 0),
    unit TEXT NOT NULL,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plan_tiers_org_id ON plan_tiers(org_id);
CREATE INDEX IF NOT EXISTS idx_plan_tiers_plan_price_id ON plan_tiers(plan_price_id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_plan_tiers_idempotency_key ON plan_tiers(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
