package videos

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns the SQL for `videos`. It always joins on `categories`
// to populate the display category name in a single query.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// videoSelect is the SELECT-list used by reads. Column order MUST match
// scanVideo below.
const videoSelect = `
    v.id, v.category_id, c.name AS category_name,
    v.title, v.slug, v.description,
    v.status, v.visibility,
    v.video_filename, v.video_original_name, v.video_mime_type, v.video_size_bytes,
    v.thumbnail_filename, v.thumbnail_second,
    v.duration_seconds, v.views_count, v.uploaded_by_admin_id,
    v.hls_status, v.hls_ready_at, v.hls_error, v.hls_variants,
    (
        SELECT COUNT(*)
        FROM site_video_publications p
        WHERE p.video_id = v.id
          AND p.deleted_at IS NULL
          AND p.status IN ('approved', 'synced')
    ) AS approved_site_count,
    v.created_at, v.updated_at, v.deleted_at
`

// InsertInput is the persistence shape for Insert — already validated.
type InsertInput struct {
	CategoryID        string
	Title             string
	Slug              string
	Description       *string
	Status            string
	Visibility        string
	VideoFilename     string
	VideoOriginalName string
	VideoMimeType     string
	VideoSizeBytes    int64
	ThumbnailFilename string
	ThumbnailSecond   int
	DurationSeconds   *int // nil → store NULL (probe failed)
	UploadedByAdminID string
}

// UpdateInput is the persistence shape for Update. nil pointers mean
// "leave unchanged".
type UpdateInput struct {
	CategoryID        *string
	Title             *string
	Slug              *string
	Description       *string
	Status            *string
	Visibility        *string
	ThumbnailFilename *string
	ThumbnailSecond   *int
	DurationSeconds   *int
}

// ListFilters is the filter set for List.
type ListFilters struct {
	Search            string
	CategoryID        string
	Status            string
	Visibility        string
	NotApprovedSiteID string
	Sort              string
	Limit             int
	Offset            int
}

// Insert creates a new video row and returns it with the joined category name.
func (r *Repository) Insert(ctx context.Context, in InsertInput) (*Video, error) {
	const insertSQL = `
        WITH inserted AS (
            INSERT INTO videos (
                category_id, title, slug, description, status, visibility,
                video_filename, video_original_name, video_mime_type, video_size_bytes,
                thumbnail_filename, thumbnail_second, duration_seconds, uploaded_by_admin_id
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
            RETURNING *
        )
        SELECT ` + videoSelect + `
        FROM inserted v
        INNER JOIN categories c ON c.id = v.category_id
    `
	row := r.pool.QueryRow(ctx, insertSQL,
		in.CategoryID,
		in.Title,
		in.Slug,
		in.Description,
		in.Status,
		in.Visibility,
		in.VideoFilename,
		in.VideoOriginalName,
		in.VideoMimeType,
		in.VideoSizeBytes,
		in.ThumbnailFilename,
		in.ThumbnailSecond,
		in.DurationSeconds,
		in.UploadedByAdminID,
	)
	v, err := scanVideo(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("insert video: %w", err)
	}
	return v, nil
}

// FindByID returns the non-deleted video matching id, with category name joined.
func (r *Repository) FindByID(ctx context.Context, id string) (*Video, error) {
	const query = `
        SELECT ` + videoSelect + `
        FROM videos v
        INNER JOIN categories c ON c.id = v.category_id
        WHERE v.id = $1 AND v.deleted_at IS NULL
        LIMIT 1
    `
	row := r.pool.QueryRow(ctx, query, id)
	v, err := scanVideo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find video by id: %w", err)
	}
	return v, nil
}

// FindBySlug returns the non-deleted video matching slug
// (case-insensitive), with category name joined.
func (r *Repository) FindBySlug(ctx context.Context, slug string) (*Video, error) {
	const query = `
        SELECT ` + videoSelect + `
        FROM videos v
        INNER JOIN categories c ON c.id = v.category_id
        WHERE LOWER(v.slug) = LOWER($1) AND v.deleted_at IS NULL
        LIMIT 1
    `
	row := r.pool.QueryRow(ctx, query, slug)
	v, err := scanVideo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find video by slug: %w", err)
	}
	return v, nil
}

// Update applies a partial update via COALESCE. Returns ErrNotFound if
// the row doesn't exist or is soft-deleted; ErrSlugTaken on collision.
func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*Video, error) {
	const updateSQL = `
        WITH updated AS (
            UPDATE videos
            SET category_id        = COALESCE($2,  category_id),
                title              = COALESCE($3,  title),
                slug               = COALESCE($4,  slug),
                description        = COALESCE($5,  description),
                status             = COALESCE($6,  status),
                visibility         = COALESCE($7,  visibility),
                thumbnail_filename = COALESCE($8,  thumbnail_filename),
                thumbnail_second   = COALESCE($9,  thumbnail_second),
                duration_seconds   = COALESCE($10, duration_seconds),
                updated_at         = NOW()
            WHERE id = $1 AND deleted_at IS NULL
            RETURNING *
        )
        SELECT ` + videoSelect + `
        FROM updated v
        INNER JOIN categories c ON c.id = v.category_id
    `
	row := r.pool.QueryRow(ctx, updateSQL,
		id,
		in.CategoryID,
		in.Title,
		in.Slug,
		in.Description,
		in.Status,
		in.Visibility,
		in.ThumbnailFilename,
		in.ThumbnailSecond,
		in.DurationSeconds,
	)
	v, err := scanVideo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("update video: %w", err)
	}
	return v, nil
}

// SoftDelete sets deleted_at, status='deleted', updated_at — without
// touching the stored files. Returns ErrNotFound if the row is missing
// or already deleted.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const query = `
        UPDATE videos
        SET deleted_at = NOW(),
            status     = 'deleted',
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft delete video: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns a page of non-deleted videos and the total matching count.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]Video, int, error) {
	orderClause := orderClauseFor(f.Sort)

	query := `
        SELECT ` + videoSelect + `, COUNT(*) OVER() AS total_count
        FROM videos v
        INNER JOIN categories c ON c.id = v.category_id
        WHERE v.deleted_at IS NULL
          AND ($1::text IS NULL OR (v.title ILIKE $1 OR v.slug ILIKE $1))
          AND ($2::uuid IS NULL OR v.category_id = $2)
          AND ($3::text IS NULL OR v.status     = $3)
          AND ($4::text IS NULL OR v.visibility = $4)
          AND (
              $5::uuid IS NULL OR NOT EXISTS (
                  SELECT 1
                  FROM site_video_publications p
                  WHERE p.site_id = $5
                    AND p.video_id = v.id
                    AND p.deleted_at IS NULL
                    AND p.status IN ('approved', 'synced')
              )
          )
        ` + orderClause + `
        LIMIT $6 OFFSET $7
    `

	var (
		searchArg, statusArg, visArg, catArg, notApprovedSiteArg any
	)
	if s := strings.TrimSpace(f.Search); s != "" {
		searchArg = "%" + s + "%"
	}
	if f.CategoryID != "" {
		catArg = f.CategoryID
	}
	if f.Status != "" {
		statusArg = f.Status
	}
	if f.Visibility != "" {
		visArg = f.Visibility
	}
	if f.NotApprovedSiteID != "" {
		notApprovedSiteArg = f.NotApprovedSiteID
	}

	rows, err := r.pool.Query(ctx, query, searchArg, catArg, statusArg, visArg, notApprovedSiteArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list videos: %w", err)
	}
	defer rows.Close()

	items := []Video{}
	total := 0
	for rows.Next() {
		var v Video
		var rowTotal int
		if err := rows.Scan(
			&v.ID, &v.CategoryID, &v.CategoryName,
			&v.Title, &v.Slug, &v.Description,
			&v.Status, &v.Visibility,
			&v.VideoFilename, &v.VideoOriginalName, &v.VideoMimeType, &v.VideoSizeBytes,
			&v.ThumbnailFilename, &v.ThumbnailSecond,
			&v.DurationSeconds, &v.ViewsCount, &v.UploadedByAdminID,
			&v.HLSStatus, &v.HLSReadyAt, &v.HLSError, &v.HLSVariants,
			&v.ApprovedSiteCount,
			&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan video row: %w", err)
		}
		items = append(items, v)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate videos: %w", err)
	}
	return items, total, nil
}

// ListPublicFilters constrains a public list query. Status='ready',
// visibility='public', and non-deleted are hardcoded — those are not
// caller-overridable.
type ListPublicFilters struct {
	Search     string
	CategoryID string // optional UUID
	Sort       string
	Limit      int
	Offset     int
}

// ListPublic returns a page of public, ready videos and the total
// matching count, in a single query via COUNT(*) OVER().
func (r *Repository) ListPublic(ctx context.Context, f ListPublicFilters) ([]Video, int, error) {
	orderClause := orderClauseFor(f.Sort)

	query := `
        SELECT ` + videoSelect + `, COUNT(*) OVER() AS total_count
        FROM videos v
        INNER JOIN categories c ON c.id = v.category_id
        WHERE v.deleted_at IS NULL
          AND v.status     = 'ready'
          AND v.visibility = 'public'
          AND c.deleted_at IS NULL
          AND ($1::text IS NULL OR (v.title ILIKE $1 OR v.slug ILIKE $1))
          AND ($2::uuid IS NULL OR v.category_id = $2)
        ` + orderClause + `
        LIMIT $3 OFFSET $4
    `

	var searchArg, catArg any
	if s := strings.TrimSpace(f.Search); s != "" {
		searchArg = "%" + s + "%"
	}
	if f.CategoryID != "" {
		catArg = f.CategoryID
	}

	rows, err := r.pool.Query(ctx, query, searchArg, catArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list public videos: %w", err)
	}
	defer rows.Close()

	items := []Video{}
	total := 0
	for rows.Next() {
		var v Video
		var rowTotal int
		if err := rows.Scan(
			&v.ID, &v.CategoryID, &v.CategoryName,
			&v.Title, &v.Slug, &v.Description,
			&v.Status, &v.Visibility,
			&v.VideoFilename, &v.VideoOriginalName, &v.VideoMimeType, &v.VideoSizeBytes,
			&v.ThumbnailFilename, &v.ThumbnailSecond,
			&v.DurationSeconds, &v.ViewsCount, &v.UploadedByAdminID,
			&v.HLSStatus, &v.HLSReadyAt, &v.HLSError, &v.HLSVariants,
			&v.ApprovedSiteCount,
			&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan public video row: %w", err)
		}
		items = append(items, v)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate public videos: %w", err)
	}
	return items, total, nil
}

// RandomPublic returns up to `limit` public, ready videos in a random
// order. If `seedKey` is non-empty, the order is deterministic for the
// (videoID, seedKey) pair — useful for per-site daily feeds. If empty,
// Postgres' `random()` is used, giving fresh ordering on each call.
//
// The query naturally deduplicates (each row appears once), so pagination
// within a single response is safe by construction.
func (r *Repository) RandomPublic(ctx context.Context, limit int, seedKey string) ([]Video, error) {
	base := `
        SELECT ` + videoSelect + `
        FROM videos v
        INNER JOIN categories c ON c.id = v.category_id
        WHERE v.deleted_at IS NULL
          AND v.status     = 'ready'
          AND v.visibility = 'public'
          AND c.deleted_at IS NULL
    `

	var (
		query string
		args  []any
	)
	if seedKey != "" {
		query = base + ` ORDER BY md5(v.id::text || $1) LIMIT $2 `
		args = []any{seedKey, limit}
	} else {
		query = base + ` ORDER BY random() LIMIT $1 `
		args = []any{limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("random public videos: %w", err)
	}
	defer rows.Close()

	items := []Video{}
	for rows.Next() {
		var v Video
		if err := rows.Scan(
			&v.ID, &v.CategoryID, &v.CategoryName,
			&v.Title, &v.Slug, &v.Description,
			&v.Status, &v.Visibility,
			&v.VideoFilename, &v.VideoOriginalName, &v.VideoMimeType, &v.VideoSizeBytes,
			&v.ThumbnailFilename, &v.ThumbnailSecond,
			&v.DurationSeconds, &v.ViewsCount, &v.UploadedByAdminID,
			&v.HLSStatus, &v.HLSReadyAt, &v.HLSError, &v.HLSVariants,
			&v.ApprovedSiteCount,
			&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan random video row: %w", err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate random videos: %w", err)
	}
	return items, nil
}

// CategoryExists reports whether a non-deleted category with the given id exists.
func (r *Repository) CategoryExists(ctx context.Context, categoryID string) (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1 FROM categories
            WHERE id = $1 AND deleted_at IS NULL
        )
    `
	var exists bool
	if err := r.pool.QueryRow(ctx, query, categoryID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check category existence: %w", err)
	}
	return exists, nil
}

// orderClauseFor returns a static ORDER BY clause from a whitelist.
func orderClauseFor(sort string) string {
	switch sort {
	case "title":
		return "ORDER BY v.title ASC, v.created_at DESC"
	case "-title":
		return "ORDER BY v.title DESC, v.created_at DESC"
	case "created_at":
		return "ORDER BY v.created_at ASC"
	case "-created_at", "":
		return "ORDER BY v.created_at DESC"
	case "views":
		return "ORDER BY v.views_count ASC, v.created_at DESC"
	case "-views":
		return "ORDER BY v.views_count DESC, v.created_at DESC"
	default:
		return "ORDER BY v.created_at DESC"
	}
}

type scanner interface {
	Scan(dest ...any) error
}

func scanVideo(row scanner) (*Video, error) {
	var v Video
	if err := row.Scan(
		&v.ID, &v.CategoryID, &v.CategoryName,
		&v.Title, &v.Slug, &v.Description,
		&v.Status, &v.Visibility,
		&v.VideoFilename, &v.VideoOriginalName, &v.VideoMimeType, &v.VideoSizeBytes,
		&v.ThumbnailFilename, &v.ThumbnailSecond,
		&v.DurationSeconds, &v.ViewsCount, &v.UploadedByAdminID,
		&v.HLSStatus, &v.HLSReadyAt, &v.HLSError, &v.HLSVariants,
		&v.ApprovedSiteCount,
		&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &v, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
