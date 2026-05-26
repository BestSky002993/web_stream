-- 000005_create_sites.down.sql

BEGIN;

DROP INDEX IF EXISTS sites_status_idx;
DROP INDEX IF EXISTS sites_api_key_hash_uniq;
DROP INDEX IF EXISTS sites_domain_uniq;
DROP TABLE IF EXISTS sites;

COMMIT;
