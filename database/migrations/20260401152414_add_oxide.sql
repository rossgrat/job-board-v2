INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000007',
    'Oxide Computer Company',
    'atomfeed',
    '{"feed_url": "https://oxide.computer/careers/feed"}',
    'https://oxide.computer/favicon.ico'
)
ON CONFLICT DO NOTHING;
