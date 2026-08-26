INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000017',
    'Divvy',
    'workable',
    '{"account_slug": "shifttransit"}',
    '/static/favicons/divvy.ico'
)
ON CONFLICT DO NOTHING;
