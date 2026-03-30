INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES
    ('019605a0-0000-7000-8000-000000000002', 'Cloudflare', 'greenhouse', '{"board_slug": "cloudflare"}', 'https://cloudflare.com/favicon.ico'),
    ('019605a0-0000-7000-8000-000000000003', 'Proton', 'greenhouse', '{"board_slug": "proton"}', 'https://proton.me/favicon.ico'),
    ('019605a0-0000-7000-8000-000000000004', 'Samsara', 'greenhouse', '{"board_slug": "samsara"}', 'https://samsara.com/favicon.ico'),
    ('019605a0-0000-7000-8000-000000000005', 'SoundCloud', 'greenhouse', '{"board_slug": "soundcloud71"}', 'https://soundcloud.com/favicon.ico'),
    ('019605a0-0000-7000-8000-000000000006', 'Tailscale', 'greenhouse', '{"board_slug": "tailscale"}', 'https://tailscale.com/favicon.ico')
ON CONFLICT DO NOTHING;
