CREATE TABLE company (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    fetch_type TEXT NOT NULL,
    fetch_config JSONB NOT NULL,
    favicon_url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
