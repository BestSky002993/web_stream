-- 000007_add_hls_variants_to_videos.up.sql
-- Phase 14: record the list of HLS quality rungs actually rendered for
-- each video (e.g. '{360p,480p,720p}'). Nullable — pre-Phase-14 rows
-- stay NULL until they're re-processed.

BEGIN;

ALTER TABLE videos
    ADD COLUMN IF NOT EXISTS hls_variants TEXT[];

COMMIT;
