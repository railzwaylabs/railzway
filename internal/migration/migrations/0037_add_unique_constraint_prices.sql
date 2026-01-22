
CREATE UNIQUE INDEX IF NOT EXISTS
ux_prices_org_id_code ON prices(org_id, code);
