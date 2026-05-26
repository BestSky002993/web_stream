-- 000003_create_videos.down.sql

BEGIN;

DROP INDEX IF EXISTS videos_created_at_idx;
DROP INDEX IF EXISTS videos_visibility_idx;
DROP INDEX IF EXISTS videos_status_idx;
DROP INDEX IF EXISTS videos_category_id_idx;
DROP INDEX IF EXISTS videos_slug_uniq;
DROP TABLE IF EXISTS videos;

COMMIT;
