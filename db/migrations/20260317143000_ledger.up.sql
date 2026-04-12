CREATE TABLE IF NOT EXISTS ledger_accounts (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('assets', 'liability', 'income', 'expense', 'equity')),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_ledger_accounts_org_code
    ON ledger_accounts(org_id, code);
CREATE INDEX IF NOT EXISTS idx_ledger_accounts_org_id ON ledger_accounts(org_id);

CREATE TABLE IF NOT EXISTS ledger_transactions (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    currency TEXT NOT NULL,
    base_currency TEXT,
    fx_rate NUMERIC(18, 8),
    base_amount_cents BIGINT,
    source_type TEXT NOT NULL,
    source_id UUID NOT NULL,
    reference_type TEXT,
    reference_id TEXT,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    invoice_item_id UUID REFERENCES invoice_items(id) ON DELETE SET NULL,
    plan_price_id UUID REFERENCES plan_prices(id) ON DELETE SET NULL,
    plan_amount_id UUID REFERENCES plan_amounts(id) ON DELETE SET NULL,
    meter_id UUID REFERENCES meters(id) ON DELETE SET NULL,
    rating_result_id UUID REFERENCES rating_results(id) ON DELETE SET NULL,
    usage_aggregate_id UUID REFERENCES usage_aggregates(id) ON DELETE SET NULL,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    posted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE ledger_transactions
    ADD CONSTRAINT ledger_transactions_period_range
        CHECK (period_end IS NULL OR period_start IS NULL OR period_end >= period_start);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_ledger_transactions_idempotency_key
    ON ledger_transactions(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_org_id ON ledger_transactions(org_id);
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_source ON ledger_transactions(org_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_reference ON ledger_transactions(org_id, reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_invoice_id ON ledger_transactions(org_id, invoice_id);
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_customer_id ON ledger_transactions(org_id, customer_id);
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_subscription_id ON ledger_transactions(org_id, subscription_id);
CREATE INDEX IF NOT EXISTS idx_ledger_transactions_period ON ledger_transactions(org_id, period_start, period_end);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES ledger_transactions(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    account_id UUID REFERENCES ledger_accounts(id) ON DELETE SET NULL,
    account_code TEXT NOT NULL,
    entry_type TEXT NOT NULL CHECK (entry_type IN ('debit', 'credit')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL,
    description TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE ledger_entries
    ADD CONSTRAINT fk_ledger_entries_account_code
        FOREIGN KEY (org_id, account_code)
        REFERENCES ledger_accounts(org_id, code);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_org_id ON ledger_entries(org_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_transaction_id ON ledger_entries(transaction_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_code ON ledger_entries(org_id, account_code);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_id ON ledger_entries(account_id);
