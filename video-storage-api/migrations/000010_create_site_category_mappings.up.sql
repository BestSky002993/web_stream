BEGIN;

CREATE TABLE IF NOT EXISTS site_remote_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    remote_term_id TEXT NOT NULL,
    remote_slug TEXT NOT NULL,
    remote_name TEXT NOT NULL,
    remote_taxonomy TEXT NOT NULL,
    parent_remote_term_id TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT site_remote_categories_term_nonblank CHECK (length(trim(remote_term_id)) > 0),
    CONSTRAINT site_remote_categories_slug_nonblank CHECK (length(trim(remote_slug)) > 0),
    CONSTRAINT site_remote_categories_name_nonblank CHECK (length(trim(remote_name)) > 0),
    CONSTRAINT site_remote_categories_taxonomy_nonblank CHECK (length(trim(remote_taxonomy)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS site_remote_categories_site_term_taxonomy_idx
    ON site_remote_categories (site_id, remote_taxonomy, remote_term_id);

CREATE INDEX IF NOT EXISTS site_remote_categories_site_slug_idx
    ON site_remote_categories (site_id, remote_slug);

CREATE TABLE IF NOT EXISTS site_category_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    backend_category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    remote_category_id UUID NULL REFERENCES site_remote_categories(id) ON DELETE SET NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sync_mode TEXT NOT NULL DEFAULT 'block',
    created_by TEXT NULL,
    confirmed_by_wp BOOLEAN NOT NULL DEFAULT FALSE,
    confirmed_by_stream BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT site_category_mappings_sync_mode_chk CHECK (sync_mode IN ('publish', 'draft', 'block'))
);

CREATE UNIQUE INDEX IF NOT EXISTS site_category_mappings_site_backend_idx
    ON site_category_mappings (site_id, backend_category_id);

CREATE INDEX IF NOT EXISTS site_category_mappings_site_enabled_idx
    ON site_category_mappings (site_id, enabled);

CREATE INDEX IF NOT EXISTS site_category_mappings_remote_idx
    ON site_category_mappings (remote_category_id);

COMMIT;
