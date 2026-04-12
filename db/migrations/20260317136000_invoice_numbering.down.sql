DROP TABLE IF EXISTS organization_invoice_number_formats;
DROP TABLE IF EXISTS invoice_sequences;

ALTER TABLE organization_billing_preferences
    DROP COLUMN IF EXISTS invoice_prefix,
    DROP COLUMN IF EXISTS invoice_number_format,
    DROP COLUMN IF EXISTS invoice_sequence_scope;
