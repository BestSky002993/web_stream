-- 000006_add_hls_to_videos.down.sql

BEGIN;

DROP INDEX IF EXISTS videos_hls_status_idx;
ALTER TABLE videos DROP CONSTRAINT IF EXISTS videos_hls_status_chk;
ALTER TABLE videos
    DROP COLUMN IF EXISTS hls_error,
    DROP COLUMN IF EXISTS hls_ready_at,
    DROP COLUMN IF EXISTS hls_status;

COMMIT;
