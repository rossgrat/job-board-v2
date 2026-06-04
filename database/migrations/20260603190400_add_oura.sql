INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000011',
    'Oura',
    'greenhouse',
    '{"board_slug": "oura"}',
    'https://ouraring.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
