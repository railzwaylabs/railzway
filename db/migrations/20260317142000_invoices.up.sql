CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE RESTRICT,
    number TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
    currency TEXT NOT NULL,
    subtotal_cents BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_cents >= 0),
    tax_cents BIGINT NOT NULL DEFAULT 0 CHECK (tax_cents >= 0),
    total_cents BIGINT NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    amount_due_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_due_cents >= 0),
    amount_paid_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_paid_cents >= 0),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    issued_at TIMESTAMPTZ,
    due_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    voided_at TIMESTAMPTZ,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN invoices.status IS 'draft|open|paid|void|uncollectible';
COMMENT ON COLUMN invoices.currency IS 'ISO 4217 currency code (e.g., "USD", "IDR")';
COMMENT ON COLUMN invoices.subtotal_cents IS 'Sum of line item amounts before tax';
COMMENT ON COLUMN invoices.tax_cents IS 'Total tax amount';
COMMENT ON COLUMN invoices.total_cents IS 'subtotal + tax';
COMMENT ON COLUMN invoices.amount_due_cents IS 'Amount remaining to be paid';
COMMENT ON COLUMN invoices.amount_paid_cents IS 'Amount already paid';
COMMENT ON COLUMN invoices.period_start IS 'Start of the billing period covered by this invoice';
COMMENT ON COLUMN invoices.period_end IS 'End of the billing period covered by this invoice';
COMMENT ON COLUMN invoices.issued_at IS 'When the invoice was issued';
COMMENT ON COLUMN invoices.due_at IS 'When payment is due';
COMMENT ON COLUMN invoices.paid_at IS 'When the invoice was fully paid';
COMMENT ON COLUMN invoices.voided_at IS 'When the invoice was voided';
COMMENT ON COLUMN invoices.idempotency_key IS 'Client-provided dedupe key for invoice creation';

ALTER TABLE invoices
    ADD CONSTRAINT invoices_period_range
        CHECK (period_end >= period_start);

CREATE UNIQUE INDEX IF NOT EXISTS ux_invoices_org_number
    ON invoices(org_id, number);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_invoices_idempotency_key
    ON invoices(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_invoices_org_id ON invoices(org_id);
CREATE INDEX IF NOT EXISTS idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(org_id, status);

CREATE TABLE IF NOT EXISTS invoice_items (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE RESTRICT,
    plan_price_id UUID REFERENCES plan_prices(id) ON DELETE RESTRICT,
    meter_id UUID REFERENCES meters(id) ON DELETE RESTRICT,
    rating_result_id UUID REFERENCES rating_results(id) ON DELETE SET NULL,
    line_type TEXT NOT NULL CHECK (line_type IN ('subscription', 'usage', 'adjustment', 'credit', 'tax')),
    description TEXT,
    quantity NUMERIC NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    unit_amount_cents NUMERIC(28,12) NOT NULL DEFAULT 0 CHECK (unit_amount_cents >= 0),
    amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
    currency TEXT NOT NULL,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN invoice_items.line_type IS 'subscription|usage|adjustment|credit|tax';
COMMENT ON COLUMN invoice_items.quantity IS 'Quantity for usage/subscription lines; typically 1 for fixed fees';
COMMENT ON COLUMN invoice_items.unit_amount_cents IS 'Price per unit';
COMMENT ON COLUMN invoice_items.amount_cents IS 'line_amount = quantity * unit_amount_cents';
COMMENT ON COLUMN invoice_items.period_start IS 'Start of the period this line item covers';
COMMENT ON COLUMN invoice_items.period_end IS 'End of the period this line item covers';
COMMENT ON COLUMN invoice_items.idempotency_key IS 'Client-provided dedupe key for line item creation';

ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_period_range
        CHECK (period_end IS NULL OR period_start IS NULL OR period_end >= period_start);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_invoice_items_idempotency_key
    ON invoice_items(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id ON invoice_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_org_id ON invoice_items(org_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_customer_id ON invoice_items(customer_id);
