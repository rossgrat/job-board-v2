CREATE TABLE company (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    fetch_type TEXT NOT NULL,
    fetch_config JSONB NOT NULL,
    favicon_url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE raw_job (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES company(id),
    source_job_id TEXT NOT NULL,
    url TEXT NOT NULL,
    raw_data JSONB NOT NULL,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_status TEXT,
    UNIQUE (company_id, source_job_id)
);

CREATE TABLE classified_job (
    id UUID PRIMARY KEY,
    raw_job_id UUID NOT NULL REFERENCES raw_job(id),
    classification_prompt_version TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    is_current BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Normalization fields
    title TEXT,
    salary_min INTEGER,
    salary_max INTEGER,
    level TEXT,
    normalized_at TIMESTAMPTZ,

    -- Classification fields
    category TEXT,
    relevance TEXT,
    reasoning TEXT,
    classified_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX classified_job_one_current_per_raw_job
    ON classified_job (raw_job_id) WHERE is_current = true;

CREATE TABLE classified_job_location (
    id UUID PRIMARY KEY,
    classified_job_id UUID NOT NULL REFERENCES classified_job(id) ON DELETE CASCADE,
    country TEXT NOT NULL,
    city TEXT,
    setting TEXT NOT NULL
);

CREATE TABLE classified_job_technology (
    id UUID PRIMARY KEY,
    classified_job_id UUID NOT NULL REFERENCES classified_job(id) ON DELETE CASCADE,
    name TEXT NOT NULL
);

CREATE TABLE outbox_task (
    id UUID PRIMARY KEY,
    classified_job_id UUID NOT NULL REFERENCES classified_job(id) ON DELETE CASCADE,
    task_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'waiting',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    not_before TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
