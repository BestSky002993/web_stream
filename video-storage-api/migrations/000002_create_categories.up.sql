-- 000002_create_categories.up.sql
-- Creates the categories table used to organize videos.

BEGIN;

CREATE TABLE IF NOT EXISTS categories (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT         NOT NULL,
    slug         TEXT         NOT NULL,
    description  TEXT,
    status       TEXT         NOT NULL DEFAULT 'active',
    sort_order   INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,

    CONSTRAINT categories_status_chk
        CHECK (status IN ('active', 'disabled')),

    CONSTRAINT categories_name_nonblank
        CHECK (length(trim(name)) > 0),

    CONSTRAINT categories_slug_nonblank
        CHECK (length(trim(slug)) > 0)
);

-- Slug uniqueness only among live rows so a deleted category's slug
-- can be reused.
CREATE UNIQUE INDEX IF NOT EXISTS categories_slug_uniq
    ON categories (LOWER(slug))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS categories_status_idx
    ON categories (status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS categories_sort_order_idx
    ON categories (sort_order, created_at DESC)
    WHERE deleted_at IS NULL;

COMMIT;
