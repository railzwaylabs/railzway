-- Payment Method Configurations
-- Stores available payment methods and their routing rules
CREATE TABLE payment_method_configs (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    
    -- Method definition
    method_type VARCHAR(50) NOT NULL, -- 'card', 'virtual_account', 'ewallet'
    method_name VARCHAR(100) NOT NULL, -- 'card_global', 'va_bca', 'gopay'
    
    -- Availability rules (JSON)
    availability_rules JSONB NOT NULL DEFAULT '{}',
    -- Example: {"countries": ["ID", "PH"], "currencies": ["IDR", "PHP"], "min_amount": 10000}
    
    -- Provider routing
    provider VARCHAR(50) NOT NULL, -- 'xendit', 'stripe'
    provider_method_type VARCHAR(50), -- Provider-specific type (e.g., 'BCA', 'GOPAY')
    
    -- Display metadata
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    icon_url VARCHAR(255),
    
    -- Priority for selection (higher = shown first)
    priority INT DEFAULT 0,
    
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_method_name_per_org UNIQUE(org_id, method_name)
);

CREATE INDEX idx_payment_method_configs_org ON payment_method_configs(org_id);
CREATE INDEX idx_payment_method_configs_active ON payment_method_configs(org_id, is_active) WHERE is_active = true;
CREATE INDEX idx_payment_method_configs_provider ON payment_method_configs(provider);

-- Customer Payment Methods
-- Stores tokenized payment methods for customers
CREATE TABLE customer_payment_methods (
    id BIGINT PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    
    -- Customer-facing type (abstracted from provider)
    type VARCHAR(50) NOT NULL, -- 'card', 'virtual_account', 'ewallet'
    
    -- Provider details (internal)
    provider VARCHAR(50) NOT NULL, -- 'xendit', 'stripe'
    provider_payment_method_id VARCHAR(255) NOT NULL, -- Token: 'pm_xxx', 'multi_use_token_id'
    
    -- Display metadata (NOT sensitive data)
    last4 VARCHAR(4), -- Last 4 digits of card/account
    brand VARCHAR(50), -- 'visa', 'mastercard', 'bca', 'gopay'
    exp_month INT, -- For cards
    exp_year INT, -- For cards
    
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_provider_pm UNIQUE(provider, provider_payment_method_id)
);

CREATE INDEX idx_customer_payment_methods_customer ON customer_payment_methods(customer_id);
CREATE INDEX idx_customer_payment_methods_default ON customer_payment_methods(customer_id, is_default) WHERE is_default = true;
CREATE INDEX idx_customer_payment_methods_provider ON customer_payment_methods(provider);

-- Ensure only one default payment method per customer
CREATE UNIQUE INDEX idx_one_default_pm_per_customer 
ON customer_payment_methods(customer_id) 
WHERE is_default = true;

-- Add country and currency to customers table
ALTER TABLE customers 
ADD COLUMN IF NOT EXISTS country VARCHAR(2), -- ISO 3166-1 alpha-2 (ID, US, SG, etc.)
ADD COLUMN IF NOT EXISTS currency VARCHAR(3), -- ISO 4217 (IDR, USD, PHP, etc.)
ADD COLUMN IF NOT EXISTS billing_address JSONB,
ADD COLUMN IF NOT EXISTS tax_id VARCHAR(255);

CREATE INDEX idx_customers_country ON customers(country);
CREATE INDEX idx_customers_currency ON customers(currency);

-- (Seed data removed to avoid FK violations on fresh installs)
