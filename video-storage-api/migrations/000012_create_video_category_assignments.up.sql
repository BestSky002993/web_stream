-- 000012_create_video_category_assignments.up.sql
-- Adds many-to-many category assignments while preserving videos.category_id
-- as the legacy primary category for existing clients.

BEGIN;

CREATE TABLE IF NOT EXISTS video_category_assignments (
    video_id    UUID        NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    category_id UUID        NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (video_id, category_id)
);

INSERT INTO video_category_assignments (video_id, category_id, created_at)
SELECT id, category_id, created_at
FROM videos
WHERE deleted_at IS NULL
ON CONFLICT (video_id, category_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS video_category_assignments_category_id_idx
    ON video_category_assignments (category_id);

CREATE INDEX IF NOT EXISTS video_category_assignments_video_id_idx
    ON video_category_assignments (video_id);

COMMIT;
