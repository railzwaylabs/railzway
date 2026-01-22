ALTER TABLE subscription_entitlements ADD COLUMN org_id BIGINT;
ALTER TABLE subscription_entitlements ADD COLUMN product_id BIGINT;
CREATE INDEX idx_subscription_entitlements_org ON subscription_entitlements(org_id);
CREATE INDEX idx_subscription_entitlements_product ON subscription_entitlements(product_id);
