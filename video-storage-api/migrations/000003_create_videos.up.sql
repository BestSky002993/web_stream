-- 000003_create_videos.up.sql
-- Creates the videos table.

BEGIN;

CREATE TABLE IF NOT EXISTS videos (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id            UUID         NOT NULL,
    title                  TEXT         NOT NULL,
    slug                   TEXT         NOT NULL,
    description            TEXT,
    status                 TEXT         NOT NULL DEFAULT 'draft',
    visibility             TEXT         NOT NULL DEFAULT 'private',
    video_filename         TEXT         NOT NULL,
    video_original_name    TEXT         NOT NULL,
    video_mime_type        TEXT         NOT NULL,
    video_size_bytes       BIGINT       NOT NULL,
    thumbnail_filename     TEXT         NOT NULL,
    duration_seconds       INTEGER,
    views_count            BIGINT       NOT NULL DEFAULT 0,
    uploaded_by_admin_id   UUID         NOT NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at             TIMESTAMPTZ,

    CONSTRAINT videos_status_chk
        CHECK (status IN ('draft', 'processing', 'ready', 'blocked', 'deleted')),

    CONSTRAINT videos_visibility_chk
        CHECK (visibility IN ('public', 'private')),

    CONSTRAINT videos_title_nonblank
        CHECK (length(trim(title)) > 0),

    CONSTRAINT videos_slug_nonblank
        CHECK (length(trim(slug)) > 0),

    CONSTRAINT videos_size_nonneg
        CHECK (video_size_bytes >= 0),

    CONSTRAINT videos_views_nonneg
        CHECK (views_count >= 0),

    CONSTRAINT videos_duration_nonneg
        CHECK (duration_seconds IS NULL OR duration_seconds >= 0),

    -- The FK guarantees existence; the "not deleted" requirement is
    -- enforced in the service layer (soft-delete is application-level).
    CONSTRAINT videos_category_fk
        FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE RESTRICT,

    CONSTRAINT videos_uploader_fk
        FOREIGN KEY (uploaded_by_admin_id) REFERENCES admin_users (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS videos_slug_uniq
    ON videos (LOWER(slug))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS videos_category_id_idx
    ON videos (category_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS videos_status_idx
    ON videos (status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS videos_visibility_idx
    ON videos (visibility)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS videos_created_at_idx
    ON videos (created_at DESC)
    WHERE deleted_at IS NULL;

COMMIT;
