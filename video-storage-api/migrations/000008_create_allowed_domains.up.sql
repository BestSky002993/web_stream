BEGIN;

CREATE TABLE IF NOT EXISTS allowed_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('stream', 'cors', 'both')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS allowed_domains_domain_live_idx
    ON allowed_domains (LOWER(domain))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS allowed_domains_active_type_idx
    ON allowed_domains (type, status)
    WHERE deleted_at IS NULL;

COMMIT;
