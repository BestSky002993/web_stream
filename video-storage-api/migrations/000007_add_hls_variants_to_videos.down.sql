-- 000007_add_hls_variants_to_videos.down.sql

BEGIN;

ALTER TABLE videos DROP COLUMN IF EXISTS hls_variants;

COMMIT;
