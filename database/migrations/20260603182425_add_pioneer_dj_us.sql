UPDATE company SET name = 'Pioneer DJ (JP)' WHERE name = 'Pioneer DJ';

INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-00000000000d',
    'Pioneer DJ (US)',
    'doorsopen',
    '{"company_id": "79877"}',
    'https://www.pioneerdj.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
