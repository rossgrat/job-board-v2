INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-00000000000a',
    'Beatport',
    'smartrecruiters',
    '{"company_identifier": "Beatport"}',
    'https://www.beatport.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
