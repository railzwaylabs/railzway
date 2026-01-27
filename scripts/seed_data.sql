-- Seed Data for Railzway (Catalog + Customers + Payments)
-- Organization ID: 1 (Default)

-- ... [Global Setup] ...
INSERT INTO organizations (id, name, slug, country_code, timezone_name)
VALUES (1, 'Railzway Demo Org', 'railzway-demo', 'US', 'UTC')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, display_name, provider, external_id)
VALUES (1, 'admin@railzway.com', 'Admin User', 'email', 'auth0|123456')
ON CONFLICT (id) DO NOTHING;

-- Tax Definitions
INSERT INTO tax_definitions (id, org_id, code, name, tax_mode, rate, description)
VALUES 
(1, 1, 'VAT_STD', 'VAT Standard', 'exclusive', 0.2000, 'Standard VAT 20%'),
(2, 1, 'US_NY', 'NY Sales Tax', 'exclusive', 0.08875, 'New York Sales Tax')
ON CONFLICT (id) DO NOTHING;

-- ==========================================
-- 2. PAYMENT CONFIGURATION
-- ==========================================

-- Payment Provider Configs (Stripe)
INSERT INTO payment_provider_configs (id, org_id, provider, config, is_active)
VALUES
(1, 1, 'stripe', '{"secret_key": "sk_test_mock", "publishable_key": "pk_test_mock"}', true),
(2, 1, 'manual', '{}', true)
ON CONFLICT (id) DO NOTHING;



-- Correction based on schema reading of 0039:
-- method_type, method_name, provider, provider_method_type, display_name
INSERT INTO payment_method_configs (id, org_id, method_type, method_name, provider, provider_method_type, display_name, is_active)
VALUES
(10, 1, 'card', 'card_global', 'stripe', 'card', 'Credit / Debit Card', true),
(20, 1, 'virtual_account', 'va_manual', 'manual', 'bank_transfer', 'Bank Transfer', true)
ON CONFLICT (id) DO NOTHING;


-- ==========================================
-- 3. CATALOG
-- ==========================================
INSERT INTO meters (id, org_id, code, name, aggregation, unit, active) VALUES
(1, 1, 'api_calls', 'API Calls', 'sum', 'calls', true),
(2, 1, 'storage_gb', 'Storage', 'max', 'gigabytes', true),
(3, 1, 'active_users', 'Active Users', 'unique_count', 'users', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO products (id, org_id, code, name, description, active) VALUES
(1, 1, 'starter', 'Starter', 'For individuals', true),
(2, 1, 'pro', 'Pro', 'For growing teams', true),
(3, 1, 'enterprise', 'Enterprise', 'For large orgs', true),
(4, 1, 'storage_addon', 'Storage Add-on', 'Tiered storage pricing', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO prices (id, org_id, product_id, code, name, pricing_model, billing_mode, billing_interval, billing_interval_count, tax_behavior, active) VALUES 
(1, 1, 1, 'price_starter_mo', 'Starter Monthly', 'flat_fee', 'in_advance', 'month', 1, 'exclusive', true),
(2, 1, 2, 'price_pro_mo', 'Pro Monthly', 'flat_fee', 'in_advance', 'month', 1, 'exclusive', true),
(3, 1, 2, 'price_pro_usage', 'API Overage', 'per_unit', 'in_arrears', 'month', 1, 'exclusive', true),
(4, 1, 3, 'price_ent_mo', 'Enterprise Monthly', 'flat_fee', 'in_advance', 'month', 1, 'exclusive', true),
(5, 1, 4, 'price_storage_graduated', 'Graduated Storage', 'graduated', 'in_arrears', 'month', 1, 'exclusive', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO price_amounts (id, org_id, price_id, currency, unit_amount_cents, effective_from) VALUES 
(1, 1, 1, 'USD', 2900, NOW()),
(2, 1, 2, 'USD', 9900, NOW()),
(3, 1, 3, 'USD', 1, NOW()),
(4, 1, 4, 'USD', 200000, NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO price_tiers (id, org_id, price_id, tier_mode, start_quantity, end_quantity, unit_amount_cents, unit) VALUES
(1, 1, 5, 2, 0, 10, 0, 'gigabytes'),
(2, 1, 5, 2, 10, 100, 50, 'gigabytes'),
(3, 1, 5, 2, 100, NULL, 10, 'gigabytes')
ON CONFLICT (id) DO NOTHING;


-- ==========================================
-- 4. CUSTOMERS & PAYMENT METHODS
-- ==========================================
INSERT INTO customers (id, org_id, name, email) VALUES 
(1, 1, 'Acme Corp', 'billing@acme.com'),
(2, 1, 'Stark Industries', 'tony@stark.com'),
(3, 1, 'Pied Piper', 'richard@piedpiper.com'),
(4, 1, 'Hooli', 'gavin@hooli.com'),
(5, 1, 'Massive Dynamic', 'bell@massive.com'),
(6, 1, 'Boxy Inc', 'storage@boxy.com'),
(7, 1, 'Upgrade Inc', 'change@upgrade.com'),
(8, 1, 'Late Corp', 'late@late.com')
ON CONFLICT (id) DO NOTHING;

-- Seed Default Payment Methods (Cards) - REMOVED per request
-- System is configured for Stripe Card & Manual Wire, but customers have no saved methods yet.

