INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000010',
    'Whoop',
    'lever',
    '{"site": "whoop"}',
    'https://www.whoop.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
