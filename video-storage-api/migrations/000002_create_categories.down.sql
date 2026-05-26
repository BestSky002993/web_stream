-- 000002_create_categories.down.sql
-- Reverses 000002_create_categories.up.sql.

BEGIN;

DROP INDEX IF EXISTS categories_sort_order_idx;
DROP INDEX IF EXISTS categories_status_idx;
DROP INDEX IF EXISTS categories_slug_uniq;
DROP TABLE IF EXISTS categories;

COMMIT;
