INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES (
    '019605a0-0000-7000-8000-000000000008',
    'Nvidia',
    'workday',
    '{"base_url": "https://nvidia.wd5.myworkdayjobs.com", "tenant": "nvidia", "site": "NVIDIAExternalCareerSite"}',
    'https://nvidia.com/favicon.ico'
)
ON CONFLICT DO NOTHING;
