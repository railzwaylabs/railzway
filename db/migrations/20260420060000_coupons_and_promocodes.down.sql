ALTER TABLE invoice_items
    DROP CONSTRAINT IF EXISTS invoice_items_line_type_check;

ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_line_type_check
        CHECK (line_type IN ('subscription', 'usage', 'adjustment', 'credit', 'tax'));

COMMENT ON COLUMN invoice_items.line_type IS 'subscription|usage|adjustment|credit|tax';

DROP TABLE IF EXISTS subscription_coupons;
DROP TABLE IF EXISTS promotion_codes;
DROP TABLE IF EXISTS coupons;
