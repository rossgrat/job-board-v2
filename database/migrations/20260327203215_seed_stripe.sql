INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000001',
    'Stripe',
    'greenhouse',
    '{"board_slug": "stripe"}',
    'https://stripe.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
