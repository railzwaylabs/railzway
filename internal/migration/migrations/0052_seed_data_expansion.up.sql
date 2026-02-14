INSERT INTO integration_catalog (id, type, name, description, logo_url, auth_type, schema, is_active)
VALUES (
    'clickhouse',
    'data_warehouse',
    'ClickHouse',
    'High-performance columnar database for real-time analytics and big data ingestion.',
    'https://cdn.simpleicons.org/clickhouse',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "host": { "type": "string", "title": "Host" },
            "port": { "type": "string", "title": "Port" },
            "username": { "type": "string", "title": "Username" },
            "password": { "type": "string", "title": "Password" },
            "database": { "type": "string", "title": "Database Name" }
        },
        "required": ["host", "port", "username", "password"]
    }'::jsonb,
    true
),
(
    'bigquery',
    'data_warehouse',
    'Google BigQuery',
    'Serverless, highly scalable, and cost-effective multi-cloud data warehouse.',
    'https://cdn.simpleicons.org/googlebigquery',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "project_id": { "type": "string", "title": "Project ID" },
            "dataset_id": { "type": "string", "title": "Dataset ID" },
            "credentials_json": { "type": "string", "title": "Service Account JSON", "format": "textarea" }
        },
        "required": ["project_id", "dataset_id", "credentials_json"]
    }'::jsonb,
    true
),
(
    'metabase',
    'analytics',
    'Metabase',
    'The simplest, fastest way to get business intelligence and analytics to everyone in your company.',
    'https://cdn.simpleicons.org/metabase',
    'api_key',
    '{
        "type": "object",
        "properties": {
            "site_url": { "type": "string", "title": "Metabase Site URL" },
            "api_key": { "type": "string", "title": "API Key" }
        },
        "required": ["site_url", "api_key"]
    }'::jsonb,
    true
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    logo_url = EXCLUDED.logo_url,
    schema = EXCLUDED.schema;
