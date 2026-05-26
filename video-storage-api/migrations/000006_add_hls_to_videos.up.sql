-- 000006_add_hls_to_videos.up.sql
-- Adds HLS pipeline state to existing video rows. The pipeline runs
-- alongside the MP4 streaming layer — both can serve playback.

BEGIN;

ALTER TABLE videos
    ADD COLUMN IF NOT EXISTS hls_status   TEXT         NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS hls_ready_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS hls_error    TEXT;

ALTER TABLE videos
    ADD CONSTRAINT videos_hls_status_chk
    CHECK (hls_status IN ('pending', 'processing', 'ready', 'failed', 'skipped'));

CREATE INDEX IF NOT EXISTS videos_hls_status_idx
    ON videos (hls_status)
    WHERE deleted_at IS NULL;

COMMIT;
