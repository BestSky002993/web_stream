BEGIN;

ALTER TABLE site_video_publications
    DROP CONSTRAINT IF EXISTS site_video_publications_status_check;

ALTER TABLE site_video_publications
    ADD CONSTRAINT site_video_publications_status_check
    CHECK (status IN ('draft', 'approved', 'synced', 'disabled', 'failed', 'blocked'));

ALTER TABLE site_video_publications
    ADD COLUMN IF NOT EXISTS sync_mode TEXT NULL;

ALTER TABLE site_video_publications
    DROP CONSTRAINT IF EXISTS site_video_publications_sync_mode_check;

ALTER TABLE site_video_publications
    ADD CONSTRAINT site_video_publications_sync_mode_check
    CHECK (sync_mode IS NULL OR sync_mode IN ('publish', 'draft', 'block'));

COMMIT;
