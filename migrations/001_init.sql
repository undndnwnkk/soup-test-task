CREATE TABLE IF NOT EXISTS jobs(
    id BIGSERIAL PRIMARY KEY,
    idempotency_key TEXT UNIQUE NOT NULL,
    request_hash TEXT,
    payload JSONB NOT NULL,
    state TEXT NOT NULL DEFAULT 'queued' CHECK(state IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    attempt INT DEFAULT 0 NOT NULL,
    worker_id TEXT,
    lease_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS job_effects(
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT REFERENCES jobs(id) NOT NULL,
    attempt INT NOT NULL,
    worker_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE INDEX IF NOT EXISTS state_created_at ON jobs(state, created_at);
CREATE INDEX IF NOT EXISTS job_effects_job_id ON job_effects(job_id);