CREATE TABLE IF NOT EXISTS apps_catalog (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    provider TEXT NOT NULL,
    description TEXT NOT NULL,
    capabilities JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active',
    version TEXT NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_apps_catalog_category_provider
    ON apps_catalog(category, provider);

INSERT INTO apps_catalog (id, name, category, provider, description, capabilities, status, version)
VALUES
    ('payment.stripe', 'Stripe', 'payment', 'stripe', 'Global card payments and invoicing.', '["payment_intents","refunds","webhooks"]', 'coming_soon', 'v1'),
    ('payment.xendit', 'Xendit', 'payment', 'xendit', 'SEA payment gateway (VA, e-wallet, cards).', '["charge","refunds","webhooks"]', 'coming_soon', 'v1'),
    ('payment.midtrans', 'Midtrans', 'payment', 'midtrans', 'Indonesia payments (cards, bank transfer, e-wallet).', '["charge","refunds","webhooks"]', 'coming_soon', 'v1'),
    ('email.smtp', 'SMTP', 'email', 'smtp', 'Transactional email via SMTP.', '["send_invoice","send_receipt"]', 'coming_soon', 'v1'),
    ('email.sendgrid', 'SendGrid', 'email', 'sendgrid', 'Email delivery via SendGrid.', '["send_invoice","send_receipt","webhooks"]', 'coming_soon', 'v1')
ON CONFLICT (id) DO NOTHING;
