INSERT INTO integration_catalog (id, type, name, description, logo_url, auth_type, schema, is_active)
VALUES (
    'slack',
    'notification',
    'Slack',
    'Get notified when invoices are finalized, payments fail, or subscriptions are canceled.',
    'https://cdn.simpleicons.org/slack',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "webhook_url": { "type": "string", "title": "Slack Webhook URL" },
            "channel_name": { "type": "string", "title": "Channel Name" }
        },
        "required": ["webhook_url"]
    }'::jsonb,
    true
),
(
    'discord',
    'notification',
    'Discord',
    'Receive billing alerts and automated reports in your Discord server.',
    'https://cdn.simpleicons.org/discord',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "webhook_url": { "type": "string", "title": "Discord Webhook URL" }
        },
        "required": ["webhook_url"]
    }'::jsonb,
    true
),
(
    'xero',
    'accounting',
    'Xero',
    'Sync your Railzway invoices and payments directly to Xero accounting.',
    'https://cdn.simpleicons.org/xero',
    'oauth2',
    '{
        "type": "object",
        "properties": {
            "tenant_id": { "type": "string", "title": "Xero Tenant ID" }
        }
    }'::jsonb,
    true
),
(
    'stripe',
    'payment',
    'Stripe',
    'Process payments and manage subscriptions using the world''s most powerful payment platform.',
    'https://cdn.simpleicons.org/stripe',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "publishable_key": { "type": "string", "title": "Publishable Key" },
            "secret_key": { "type": "string", "title": "Secret Key" },
            "webhook_secret": { "type": "string", "title": "Webhook Signing Secret" }
        },
        "required": ["publishable_key", "secret_key"]
    }'::jsonb,
    true
),
(
    'quickbooks',
    'accounting',
    'QuickBooks Online',
    'Automatically sync your invoices, customers, and payments to QuickBooks.',
    'https://cdn.simpleicons.org/quickbooks',
    'oauth2',
    '{
        "type": "object",
        "properties": {
            "company_id": { "type": "string", "title": "QuickBooks Company ID" }
        }
    }'::jsonb,
    true
),
(
    'netsuite',
    'accounting',
    'NetSuite',
    'Enterprise ERP integration for complex billing and revenue recognition.',
    'https://upload.wikimedia.org/wikipedia/commons/thumb/1/17/2024_NetSuite_logo.png/250px-2024_NetSuite_logo.png',
    'oauth2',
    '{
        "type": "object",
        "properties": {
            "account_id": { "type": "string", "title": "NetSuite Account ID" }
        }
    }'::jsonb,
    true
),
(
    'avalara',
    'tax',
    'Avalara',
    'Real-time tax calculation and compliance for global sales.',
    'https://companieslogo.com/img/orig/AVLR-c064b900.png?t=1720244490',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "account_id": { "type": "string", "title": "Account ID" },
            "license_key": { "type": "string", "title": "License Key" },
            "company_code": { "type": "string", "title": "Company Code" }
        },
        "required": ["account_id", "license_key"]
    }'::jsonb,
    true
),
(
    'salesforce',
    'crm',
    'Salesforce',
    'Keep your CRM in sync with billing data. Automate entitlement provisioning.',
    'https://cdn.simpleicons.org/salesforce',
    'oauth2',
    '{
        "type": "object",
        "properties": {
            "instance_url": { "type": "string", "title": "Salesforce Instance URL" }
        }
    }'::jsonb,
    true
),
(
    'hubspot',
    'crm',
    'HubSpot',
    'Sync deals and customers between Railzway and HubSpot.',
    'https://cdn.simpleicons.org/hubspot',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "access_token": { "type": "string", "title": "Private App Access Token" }
        },
        "required": ["access_token"]
    }'::jsonb,
    true
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    logo_url = EXCLUDED.logo_url,
    schema = EXCLUDED.schema;
