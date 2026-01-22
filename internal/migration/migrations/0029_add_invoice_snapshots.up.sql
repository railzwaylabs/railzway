CREATE TABLE invoice_tax_lines (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    invoice_id BIGINT NOT NULL,
    tax_code TEXT,
    tax_name TEXT NOT NULL,
    tax_mode TEXT NOT NULL,
    tax_rate DOUBLE PRECISION NOT NULL,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoice_tax_lines_invoice ON invoice_tax_lines(invoice_id);
CREATE INDEX idx_invoice_tax_lines_org ON invoice_tax_lines(org_id);

ALTER TABLE rating_results ADD COLUMN feature_code TEXT;
