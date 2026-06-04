INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000012',
    'Garmin',
    'jibe',
    '{"host": "careers.garmin.com"}',
    'https://www.garmin.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
