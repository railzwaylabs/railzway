CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- PERCENT, FIXED
    amount_cents BIGINT,
    percentage DOUBLE PRECISION,
    duration TEXT NOT NULL, -- ONCE, REPEATING, FOREVER
    duration_months INT,
    currency TEXT,
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    auto_apply BOOLEAN NOT NULL DEFAULT FALSE,
    target_segment TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_coupons_org_id ON coupons(org_id);
CREATE INDEX IF NOT EXISTS idx_coupons_org_auto_apply ON coupons(org_id, auto_apply);
CREATE INDEX IF NOT EXISTS idx_coupons_org_target_segment ON coupons(org_id, target_segment);

CREATE TABLE IF NOT EXISTS promotion_codes (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    max_redemptions INT,
    redemption_count INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_promotion_codes_org_code ON promotion_codes(org_id, code);
CREATE INDEX IF NOT EXISTS idx_promotion_codes_org_id ON promotion_codes(org_id);
CREATE INDEX IF NOT EXISTS idx_promotion_codes_coupon_id ON promotion_codes(coupon_id);

CREATE TABLE IF NOT EXISTS subscription_coupons (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_subscription_coupons_org_sub ON subscription_coupons(org_id, subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_coupons_org_id ON subscription_coupons(org_id);
CREATE INDEX IF NOT EXISTS idx_subscription_coupons_sub_id ON subscription_coupons(subscription_id);

ALTER TABLE invoice_items
    DROP CONSTRAINT IF EXISTS invoice_items_line_type_check;

ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_line_type_check
        CHECK (line_type IN ('subscription', 'usage', 'discount', 'adjustment', 'credit', 'tax'));

COMMENT ON COLUMN invoice_items.line_type IS 'subscription|usage|discount|adjustment|credit|tax';
