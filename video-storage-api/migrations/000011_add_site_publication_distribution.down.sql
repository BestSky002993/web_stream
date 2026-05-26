BEGIN;

ALTER TABLE site_video_publications
    DROP CONSTRAINT IF EXISTS site_video_publications_sync_mode_check;

ALTER TABLE site_video_publications
    DROP COLUMN IF EXISTS sync_mode;

ALTER TABLE site_video_publications
    DROP CONSTRAINT IF EXISTS site_video_publications_status_check;

ALTER TABLE site_video_publications
    ADD CONSTRAINT site_video_publications_status_check
    CHECK (status IN ('draft', 'approved', 'synced', 'disabled', 'failed'));

COMMIT;
