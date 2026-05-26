BEGIN;

DROP INDEX IF EXISTS allowed_domains_active_type_idx;
DROP INDEX IF EXISTS allowed_domains_domain_live_idx;
DROP TABLE IF EXISTS allowed_domains;

COMMIT;
