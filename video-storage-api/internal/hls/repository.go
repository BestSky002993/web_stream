package hls

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns just the HLS columns on the videos table:
// `hls_status`, `hls_ready_at`, `hls_error`. Everything else (video
// metadata, soft-delete, etc.) stays with the videos package.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SetProcessing marks the row as currently being converted. Clears
// any previous error so the dashboard doesn't show stale state.
func (r *Repository) SetProcessing(ctx context.Context, videoID string) error {
	const q = `
        UPDATE videos
        SET hls_status = 'processing',
            hls_error  = NULL,
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	if _, err := r.pool.Exec(ctx, q, videoID); err != nil {
		return fmt.Errorf("set hls processing: %w", err)
	}
	return nil
}

// SetReady marks conversion as complete, stamps hls_ready_at, and
// records the list of generated variants so the dashboard can render
// "Qualities: 360p, 480p, 720p" without scanning the filesystem.
func (r *Repository) SetReady(ctx context.Context, videoID string, variants []string) error {
	const q = `
        UPDATE videos
        SET hls_status   = 'ready',
            hls_ready_at = NOW(),
            hls_error    = NULL,
            hls_variants = $2,
            updated_at   = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	if _, err := r.pool.Exec(ctx, q, videoID, variants); err != nil {
		return fmt.Errorf("set hls ready: %w", err)
	}
	return nil
}

// SetFailed records the error message (truncated to 500 chars to keep
// the column manageable) so admins can debug without tailing logs.
func (r *Repository) SetFailed(ctx context.Context, videoID, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	const q = `
        UPDATE videos
        SET hls_status = 'failed',
            hls_error  = $2,
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	if _, err := r.pool.Exec(ctx, q, videoID, errMsg); err != nil {
		return fmt.Errorf("set hls failed: %w", err)
	}
	return nil
}
