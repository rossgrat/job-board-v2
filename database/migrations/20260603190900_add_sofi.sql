INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000016',
    'SoFi',
    'greenhouse',
    '{"board_slug": "sofi"}',
    'https://www.sofi.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
