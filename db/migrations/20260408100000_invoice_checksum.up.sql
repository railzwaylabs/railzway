ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN invoices.checksum IS 'SHA256 checksum of invoice core fields and line items';
