-- 000005_create_sites.up.sql
-- External websites authorised to use the video API (e.g. WordPress
-- installations). Each row owns one domain and one API key.

BEGIN;

CREATE TABLE IF NOT EXISTS sites (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT         NOT NULL,
    domain                TEXT         NOT NULL,
    api_key_hash          TEXT         NOT NULL,
    status                TEXT         NOT NULL DEFAULT 'active',
    daily_request_limit   INTEGER      NOT NULL DEFAULT 10000,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,

    CONSTRAINT sites_status_chk
        CHECK (status IN ('active', 'disabled', 'suspended')),

    CONSTRAINT sites_name_nonblank
        CHECK (length(trim(name)) > 0),

    CONSTRAINT sites_domain_nonblank
        CHECK (length(trim(domain)) > 0),

    CONSTRAINT sites_daily_limit_nonneg
        CHECK (daily_request_limit >= 0)
);

-- Domain unique among live sites (case-insensitive).
CREATE UNIQUE INDEX IF NOT EXISTS sites_domain_uniq
    ON sites (LOWER(domain))
    WHERE deleted_at IS NULL;

-- API key hash unique among live sites + indexed for the per-request
-- lookup on every site-protected endpoint.
CREATE UNIQUE INDEX IF NOT EXISTS sites_api_key_hash_uniq
    ON sites (api_key_hash)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS sites_status_idx
    ON sites (status)
    WHERE deleted_at IS NULL;

COMMIT;
