-- 000001_create_admin_users.up.sql
-- Creates the admin_users table for the admin panel.
-- Phase 2: schema only. Login logic lands in Phase 3.

BEGIN;

CREATE TABLE IF NOT EXISTS admin_users (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email                    TEXT         NOT NULL,
    password_hash            TEXT         NOT NULL,
    display_name             TEXT         NOT NULL,
    role                     TEXT         NOT NULL DEFAULT 'admin',
    status                   TEXT         NOT NULL DEFAULT 'active',
    last_login_at            TIMESTAMPTZ,
    failed_login_attempts    INTEGER      NOT NULL DEFAULT 0,
    locked_until             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ,

    CONSTRAINT admin_users_role_chk
        CHECK (role IN ('super_admin', 'admin', 'editor')),

    CONSTRAINT admin_users_status_chk
        CHECK (status IN ('active', 'disabled', 'locked')),

    CONSTRAINT admin_users_failed_login_attempts_nonneg
        CHECK (failed_login_attempts >= 0)
);

-- Case-insensitive uniqueness on email, ignoring soft-deleted rows so an
-- email can be reused if the previous account was deleted.
CREATE UNIQUE INDEX IF NOT EXISTS admin_users_email_uniq
    ON admin_users (LOWER(email))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS admin_users_status_idx
    ON admin_users (status)
    WHERE deleted_at IS NULL;

COMMIT;
