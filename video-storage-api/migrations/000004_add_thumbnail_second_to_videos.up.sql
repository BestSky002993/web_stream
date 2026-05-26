-- 000004_add_thumbnail_second_to_videos.up.sql
-- Adds the timestamp (in whole seconds) the thumbnail was extracted from.
-- thumbnail_filename already exists; only the second-offset is new.

BEGIN;

ALTER TABLE videos
    ADD COLUMN IF NOT EXISTS thumbnail_second INTEGER NOT NULL DEFAULT 1;

ALTER TABLE videos
    ADD CONSTRAINT videos_thumb_second_nonneg
    CHECK (thumbnail_second >= 0);

COMMIT;
