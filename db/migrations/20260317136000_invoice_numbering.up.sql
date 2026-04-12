ALTER TABLE organization_billing_preferences
    ADD COLUMN IF NOT EXISTS invoice_prefix TEXT NOT NULL DEFAULT 'INV',
    ADD COLUMN IF NOT EXISTS invoice_number_format TEXT NOT NULL DEFAULT '{PREFIX}-{YYYY}{MM}-{SEQ:6}',
    ADD COLUMN IF NOT EXISTS invoice_sequence_scope TEXT NOT NULL DEFAULT 'org_month';

CREATE TABLE IF NOT EXISTS invoice_sequences (
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    period_key TEXT NOT NULL,
    last_value BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (org_id, period_key)
);

CREATE TABLE IF NOT EXISTS organization_invoice_number_formats (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    format TEXT NOT NULL,
    sequence_scope TEXT NOT NULL DEFAULT 'org_month',
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
