INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-00000000000b',
    'Pioneer DJ',
    'girecruit',
    '{"tenant": "alphatheta"}',
    'https://www.pioneerdj.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
