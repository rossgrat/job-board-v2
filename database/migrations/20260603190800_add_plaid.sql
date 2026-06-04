INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000015',
    'Plaid',
    'lever',
    '{"site": "plaid"}',
    'https://plaid.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
