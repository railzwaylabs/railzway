CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_products_org_code ON products(org_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS ux_products_org_idempotency ON products(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_products_org_id ON products(org_id);

CREATE TABLE IF NOT EXISTS features (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    feature_type TEXT NOT NULL,
    meter_id UUID REFERENCES meters(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_features_org_code ON features(org_id, code);
CREATE UNIQUE INDEX IF NOT EXISTS ux_features_org_idempotency ON features(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_features_org_id ON features(org_id);
CREATE INDEX IF NOT EXISTS idx_features_meter_id ON features(meter_id);

CREATE TABLE IF NOT EXISTS product_features (
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (product_id, feature_id)
);

CREATE INDEX IF NOT EXISTS idx_product_features_product_id ON product_features(product_id);
CREATE INDEX IF NOT EXISTS idx_product_features_feature_id ON product_features(feature_id);
