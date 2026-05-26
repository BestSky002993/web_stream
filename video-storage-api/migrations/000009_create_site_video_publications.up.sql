BEGIN;

CREATE TABLE IF NOT EXISTS site_video_publications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'approved', 'synced', 'disabled', 'failed')),
    remote_post_id TEXT NULL,
    remote_slug TEXT NULL,
    category_mapping TEXT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    notes TEXT NULL,
    last_synced_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS site_video_publications_site_video_live_idx
    ON site_video_publications (site_id, video_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS site_video_publications_site_status_idx
    ON site_video_publications (site_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS site_video_publications_video_idx
    ON site_video_publications (video_id)
    WHERE deleted_at IS NULL;

COMMIT;
