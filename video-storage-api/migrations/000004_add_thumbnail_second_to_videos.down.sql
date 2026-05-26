-- 000004_add_thumbnail_second_to_videos.down.sql

BEGIN;

ALTER TABLE videos DROP CONSTRAINT IF EXISTS videos_thumb_second_nonneg;
ALTER TABLE videos DROP COLUMN IF EXISTS thumbnail_second;

COMMIT;
