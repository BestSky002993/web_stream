-- 000001_create_admin_users.down.sql
-- Reverses 000001_create_admin_users.up.sql.

BEGIN;

DROP INDEX IF EXISTS admin_users_status_idx;
DROP INDEX IF EXISTS admin_users_email_uniq;
DROP TABLE IF EXISTS admin_users;

COMMIT;
