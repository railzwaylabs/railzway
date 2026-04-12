CREATE TABLE IF NOT EXISTS tax_rates (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    percentage NUMERIC NOT NULL CHECK (percentage >= 0),
    inclusive BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_tax_rates_org_code ON tax_rates(org_id, code);
CREATE INDEX IF NOT EXISTS idx_tax_rates_org_id ON tax_rates(org_id);

CREATE TABLE IF NOT EXISTS tax_lines (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    invoice_item_id UUID REFERENCES invoice_items(id) ON DELETE SET NULL,
    tax_rate_id UUID NOT NULL REFERENCES tax_rates(id) ON DELETE RESTRICT,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tax_lines_org_id ON tax_lines(org_id);
CREATE INDEX IF NOT EXISTS idx_tax_lines_invoice_id ON tax_lines(invoice_id);
CREATE INDEX IF NOT EXISTS idx_tax_lines_tax_rate_id ON tax_lines(tax_rate_id);
