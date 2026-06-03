INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000009',
    'Mixcloud',
    'workable',
    '{"account_slug": "mixcloud-limited"}',
    'https://www.mixcloud.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
