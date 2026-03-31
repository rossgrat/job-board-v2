-- Group 1: US remote, mid/senior
INSERT INTO filter_group (id) VALUES ('019605a0-0001-7000-8000-000000000001');
INSERT INTO filter_condition (id, filter_group_id, field, operator, value) VALUES
    ('019605a0-0001-7000-8000-000000000010', '019605a0-0001-7000-8000-000000000001', 'location_country', 'equals', 'US'),
    ('019605a0-0001-7000-8000-000000000011', '019605a0-0001-7000-8000-000000000001', 'location_setting', 'equals', 'remote'),
    ('019605a0-0001-7000-8000-000000000012', '019605a0-0001-7000-8000-000000000001', 'level', 'in', '["mid", "senior"]');

-- Group 2: Chicago remote/hybrid, mid/senior
INSERT INTO filter_group (id) VALUES ('019605a0-0002-7000-8000-000000000001');
INSERT INTO filter_condition (id, filter_group_id, field, operator, value) VALUES
    ('019605a0-0002-7000-8000-000000000010', '019605a0-0002-7000-8000-000000000001', 'location_country', 'equals', 'US'),
    ('019605a0-0002-7000-8000-000000000011', '019605a0-0002-7000-8000-000000000001', 'location_city', 'equals', 'Chicago, IL'),
    ('019605a0-0002-7000-8000-000000000012', '019605a0-0002-7000-8000-000000000001', 'location_setting', 'in', '["remote", "hybrid"]'),
    ('019605a0-0002-7000-8000-000000000013', '019605a0-0002-7000-8000-000000000001', 'level', 'in', '["mid", "senior"]');
