-- 000012_create_video_category_assignments.down.sql

BEGIN;

DROP INDEX IF EXISTS video_category_assignments_video_id_idx;
DROP INDEX IF EXISTS video_category_assignments_category_id_idx;
DROP TABLE IF EXISTS video_category_assignments;

COMMIT;
