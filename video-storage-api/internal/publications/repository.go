package publications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/video-storage-api/internal/categorymappings"
	"github.com/video-storage-api/internal/videos"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type UpsertInput struct {
	SiteID          string
	VideoID         string
	Status          string
	SyncMode        *string
	RemotePostID    *string
	RemoteSlug      *string
	CategoryMapping *string
	Tags            []string
	Notes           *string
	LastSyncedAt    *time.Time
}

type BulkUpsertInput struct {
	SiteID           string
	VideoIDs         []string
	Status           string
	SyncMode         string
	UnmappedBehavior string
}

type ListFilters struct {
	SiteID  string
	VideoID string
	Status  string
	Limit   int
	Offset  int
}

const publicationColumns = `
    p.id, p.site_id, p.video_id, p.status, p.sync_mode, p.remote_post_id, p.remote_slug,
    p.category_mapping, p.tags, p.notes, p.last_synced_at,
    p.created_at, p.updated_at, p.deleted_at
`

const publicationReturningColumns = `
    id, site_id, video_id, status, sync_mode, remote_post_id, remote_slug,
    category_mapping, tags, notes, last_synced_at,
    created_at, updated_at, deleted_at
`

const publicationVideoColumns = `
    v.id, v.category_id, c.name AS category_name,
    v.title, v.slug, v.description,
    v.status, v.visibility,
    v.video_filename, v.video_original_name, v.video_mime_type, v.video_size_bytes,
    v.thumbnail_filename, v.thumbnail_second,
    v.duration_seconds, v.views_count, v.uploaded_by_admin_id,
    v.hls_status, v.hls_ready_at, v.hls_error, v.hls_variants,
    v.created_at, v.updated_at, v.deleted_at
`

func (r *Repository) Upsert(ctx context.Context, in UpsertInput) (*Publication, error) {
	const query = `
        INSERT INTO site_video_publications (
            site_id, video_id, status, sync_mode, remote_post_id, remote_slug,
            category_mapping, tags, notes, last_synced_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, '{}'::text[]), $9, $10)
        ON CONFLICT (site_id, video_id) WHERE deleted_at IS NULL
        DO UPDATE SET
            status           = EXCLUDED.status,
            sync_mode        = EXCLUDED.sync_mode,
            remote_post_id   = EXCLUDED.remote_post_id,
            remote_slug      = EXCLUDED.remote_slug,
            category_mapping = EXCLUDED.category_mapping,
            tags             = EXCLUDED.tags,
            notes            = EXCLUDED.notes,
            last_synced_at   = EXCLUDED.last_synced_at,
            updated_at       = NOW()
        RETURNING ` + publicationReturningColumns

	row := r.pool.QueryRow(ctx, query,
		in.SiteID, in.VideoID, in.Status, cleanPtr(in.SyncMode), cleanPtr(in.RemotePostID), cleanPtr(in.RemoteSlug),
		cleanPtr(in.CategoryMapping), cleanTags(in.Tags), cleanPtr(in.Notes), in.LastSyncedAt,
	)
	p, err := scanPublication(row)
	if err != nil {
		if isFKViolation(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("upsert publication: %w", err)
	}
	return p, nil
}

func (r *Repository) BulkUpsertForSite(ctx context.Context, in BulkUpsertInput) (*BulkDistributionResponse, error) {
	out := &BulkDistributionResponse{
		Errors: []BulkDistributionError{},
	}
	if len(in.VideoIDs) == 0 {
		return out, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin bulk distribution: %w", err)
	}
	defer tx.Rollback(ctx)

	var siteExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sites WHERE id = $1 AND deleted_at IS NULL)`, in.SiteID).Scan(&siteExists); err != nil {
		return nil, fmt.Errorf("check bulk distribution site: %w", err)
	}
	if !siteExists {
		return nil, ErrNotFound
	}

	const inspectQuery = `
        SELECT
            EXISTS (
                SELECT 1 FROM videos v
                WHERE v.id = $1 AND v.deleted_at IS NULL
            ) AS video_exists,
            EXISTS (
                SELECT 1 FROM site_video_publications p
                WHERE p.site_id = $2 AND p.video_id = $1 AND p.deleted_at IS NULL
            ) AS publication_exists,
            m.id IS NOT NULL AS mapping_exists,
            COALESCE(m.enabled, FALSE) AS mapping_enabled,
            COALESCE(m.sync_mode, 'block') AS mapping_sync_mode,
            COALESCE(m.confirmed_by_wp, FALSE) AS confirmed_by_wp,
            COALESCE(m.confirmed_by_stream, FALSE) AS confirmed_by_stream
        FROM videos v
        LEFT JOIN site_category_mappings m
            ON m.site_id = $2 AND m.backend_category_id = v.category_id
        WHERE v.id = $1 AND v.deleted_at IS NULL
        LIMIT 1
    `
	const upsertQuery = `
        INSERT INTO site_video_publications (site_id, video_id, status, sync_mode)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (site_id, video_id) WHERE deleted_at IS NULL
        DO UPDATE SET
            status = EXCLUDED.status,
            sync_mode = EXCLUDED.sync_mode,
            updated_at = NOW()
    `

	for _, videoID := range in.VideoIDs {
		var videoExists, publicationExists, mappingExists, mappingEnabled, confirmedByWP, confirmedByStream bool
		var mappingSyncMode string
		err := tx.QueryRow(ctx, inspectQuery, videoID, in.SiteID).Scan(
			&videoExists,
			&publicationExists,
			&mappingExists,
			&mappingEnabled,
			&mappingSyncMode,
			&confirmedByWP,
			&confirmedByStream,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				out.Skipped++
				out.Errors = append(out.Errors, BulkDistributionError{VideoID: videoID, Error: "video not found"})
				continue
			}
			return nil, fmt.Errorf("inspect bulk distribution video: %w", err)
		}
		if !videoExists {
			out.Skipped++
			out.Errors = append(out.Errors, BulkDistributionError{VideoID: videoID, Error: "video not found"})
			continue
		}

		effectiveStatus := in.Status
		effectiveSyncMode := in.SyncMode
		mappingBlocksApproval := in.Status == StatusApproved && (!mappingExists || !mappingEnabled || mappingSyncMode == categorymappings.SyncModeBlock)
		if mappingBlocksApproval {
			if in.UnmappedBehavior == "draft" {
				effectiveStatus = StatusDraft
				effectiveSyncMode = categorymappings.SyncModeDraft
				out.Warnings = append(out.Warnings, BulkDistributionWarning{VideoID: videoID, Warning: "category mapping blocks approval; saved as draft"})
			} else {
				out.Skipped++
				out.Errors = append(out.Errors, BulkDistributionError{VideoID: videoID, Error: "category mapping is missing, disabled, or blocked"})
				continue
			}
		}
		if in.Status == StatusApproved && effectiveSyncMode == categorymappings.SyncModePublish && (!confirmedByWP || !confirmedByStream) {
			if in.UnmappedBehavior == "draft" {
				effectiveStatus = StatusDraft
				effectiveSyncMode = categorymappings.SyncModeDraft
				out.Warnings = append(out.Warnings, BulkDistributionWarning{VideoID: videoID, Warning: "category mapping is unconfirmed; saved as draft"})
			} else {
				out.Skipped++
				out.Errors = append(out.Errors, BulkDistributionError{VideoID: videoID, Error: "category mapping is unconfirmed"})
				continue
			}
		}

		if _, err := tx.Exec(ctx, upsertQuery, in.SiteID, videoID, effectiveStatus, effectiveSyncMode); err != nil {
			return nil, fmt.Errorf("bulk upsert distribution: %w", err)
		}
		if effectiveStatus == StatusApproved && !publicationExists {
			out.Approved++
		} else {
			out.Updated++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit bulk distribution: %w", err)
	}
	return out, nil
}

func (r *Repository) VideoCategoryID(ctx context.Context, videoID string) (string, error) {
	const query = `SELECT category_id FROM videos WHERE id = $1 AND deleted_at IS NULL`
	var categoryID string
	if err := r.pool.QueryRow(ctx, query, videoID).Scan(&categoryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("find publication video category: %w", err)
	}
	return categoryID, nil
}

func (r *Repository) GetBySiteVideo(ctx context.Context, siteID, videoID string) (*Publication, error) {
	const query = `
        SELECT ` + publicationColumns + `
        FROM site_video_publications p
        WHERE p.site_id = $1 AND p.video_id = $2 AND p.deleted_at IS NULL
        LIMIT 1
    `
	p, err := scanPublication(r.pool.QueryRow(ctx, query, siteID, videoID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get publication by site video: %w", err)
	}
	return p, nil
}

type DistributionRow struct {
	SiteID             string
	SiteName           string
	SiteDomain         string
	SiteStatus         string
	Publication        *Publication
	MappingStatus      string
	MappingSyncMode    string
	MappingEnabled     bool
	RemoteCategoryID   *string
	RemoteCategoryName *string
}

func (r *Repository) ListSiteDistribution(ctx context.Context, videoID string) ([]DistributionRow, error) {
	categoryID, err := r.VideoCategoryID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	const query = `
        SELECT
            s.id, s.name, s.domain, s.status,
            p.id, p.site_id, p.video_id, p.status, p.sync_mode, p.remote_post_id, p.remote_slug,
            p.category_mapping, COALESCE(p.tags, '{}'::text[]), p.notes, p.last_synced_at,
            p.created_at, p.updated_at, p.deleted_at,
            m.remote_category_id, m.enabled, m.sync_mode,
            m.confirmed_by_wp, m.confirmed_by_stream,
            rc.remote_name
        FROM sites s
        LEFT JOIN site_video_publications p
            ON p.site_id = s.id AND p.video_id = $1 AND p.deleted_at IS NULL
        LEFT JOIN site_category_mappings m
            ON m.site_id = s.id AND m.backend_category_id = $2
        LEFT JOIN site_remote_categories rc
            ON rc.id = m.remote_category_id AND rc.site_id = s.id
        WHERE s.deleted_at IS NULL
        ORDER BY s.name ASC, s.domain ASC
    `
	rows, err := r.pool.Query(ctx, query, videoID, categoryID)
	if err != nil {
		return nil, fmt.Errorf("list site distribution: %w", err)
	}
	defer rows.Close()
	out := []DistributionRow{}
	for rows.Next() {
		row, err := scanDistributionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}
	return out, rows.Err()
}

func (r *Repository) SetStatus(ctx context.Context, id, status string) (*Publication, error) {
	const query = `
        UPDATE site_video_publications
        SET status = $2, updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING ` + publicationReturningColumns

	row := r.pool.QueryRow(ctx, query, id, status)
	p, err := scanPublication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("set publication status: %w", err)
	}
	return p, nil
}

func (r *Repository) UpdateSyncResult(ctx context.Context, siteID, videoID, status string, remotePostID, remoteSlug *string) (*Publication, error) {
	const query = `
        UPDATE site_video_publications
        SET status = $3,
            remote_post_id = COALESCE($4, remote_post_id),
            remote_slug = COALESCE($5, remote_slug),
            last_synced_at = CASE WHEN $3 = 'synced' THEN NOW() ELSE last_synced_at END,
            updated_at = NOW()
        WHERE site_id = $1
          AND video_id = $2
          AND deleted_at IS NULL
          AND status IN ('approved', 'synced', 'failed')
        RETURNING ` + publicationReturningColumns

	row := r.pool.QueryRow(ctx, query, siteID, videoID, status, cleanPtr(remotePostID), cleanPtr(remoteSlug))
	p, err := scanPublication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update sync result: %w", err)
	}
	return p, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const query = `
        UPDATE site_video_publications
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete publication: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Publication, int, error) {
	query := `
        SELECT ` + publicationColumns + `, ` + publicationVideoColumns + `, COUNT(*) OVER() AS total_count
        FROM site_video_publications p
        INNER JOIN videos v ON v.id = p.video_id
        INNER JOIN categories c ON c.id = v.category_id
        WHERE p.deleted_at IS NULL
          AND ($1::uuid IS NULL OR p.site_id = $1)
          AND ($2::uuid IS NULL OR p.video_id = $2)
          AND ($3::text IS NULL OR p.status = $3)
        ORDER BY p.updated_at DESC
        LIMIT $4 OFFSET $5
    `
	var siteArg, videoArg, statusArg any
	if f.SiteID != "" {
		siteArg = f.SiteID
	}
	if f.VideoID != "" {
		videoArg = f.VideoID
	}
	if f.Status != "" {
		statusArg = f.Status
	}

	rows, err := r.pool.Query(ctx, query, siteArg, videoArg, statusArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list publications: %w", err)
	}
	defer rows.Close()

	items := []Publication{}
	total := 0
	for rows.Next() {
		p, rowTotal, err := scanPublicationWithVideoAndTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate publications: %w", err)
	}
	return items, total, nil
}

func (r *Repository) ListApprovedVideos(ctx context.Context, siteID string, f videos.ListPublicFilters) ([]Publication, int, error) {
	orderClause := orderClauseFor(f.Sort)
	query := `
        SELECT ` + publicationColumns + `, ` + publicationVideoColumns + `, COUNT(*) OVER() AS total_count
        FROM site_video_publications p
        INNER JOIN videos v ON v.id = p.video_id
        INNER JOIN categories c ON c.id = v.category_id
        WHERE p.deleted_at IS NULL
          AND p.site_id = $1
          AND p.status IN ('approved', 'synced')
          AND v.deleted_at IS NULL
          AND v.status = 'ready'
          AND v.visibility = 'public'
          AND c.deleted_at IS NULL
          AND ($2::text IS NULL OR (v.title ILIKE $2 OR v.slug ILIKE $2))
          AND ($3::uuid IS NULL OR v.category_id = $3)
        ` + orderClause + `
        LIMIT $4 OFFSET $5
    `

	var searchArg, catArg any
	if s := strings.TrimSpace(f.Search); s != "" {
		searchArg = "%" + s + "%"
	}
	if f.CategoryID != "" {
		catArg = f.CategoryID
	}
	rows, err := r.pool.Query(ctx, query, siteID, searchArg, catArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list approved videos: %w", err)
	}
	defer rows.Close()

	items := []Publication{}
	total := 0
	for rows.Next() {
		p, rowTotal, err := scanPublicationWithVideoAndTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate approved videos: %w", err)
	}
	return items, total, nil
}

func (r *Repository) IsApprovedForSite(ctx context.Context, siteID, videoID string) (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1
            FROM site_video_publications p
            INNER JOIN videos v ON v.id = p.video_id
            WHERE p.deleted_at IS NULL
              AND p.site_id = $1
              AND p.video_id = $2
              AND p.status IN ('approved', 'synced')
              AND v.deleted_at IS NULL
              AND v.status = 'ready'
              AND v.visibility = 'public'
        )
    `
	var ok bool
	if err := r.pool.QueryRow(ctx, query, siteID, videoID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check site publication: %w", err)
	}
	return ok, nil
}

func (r *Repository) IsPlayableForSite(ctx context.Context, siteID, videoID string) (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1
            FROM site_video_publications p
            INNER JOIN videos v ON v.id = p.video_id
            INNER JOIN site_category_mappings m
                ON m.site_id = p.site_id
               AND m.backend_category_id = v.category_id
            WHERE p.deleted_at IS NULL
              AND p.site_id = $1
              AND p.video_id = $2
              AND p.status IN ('approved', 'synced')
              AND v.deleted_at IS NULL
              AND v.status = 'ready'
              AND v.visibility = 'public'
              AND m.enabled = TRUE
              AND COALESCE(p.sync_mode, m.sync_mode) <> 'block'
        )
    `
	var ok bool
	if err := r.pool.QueryRow(ctx, query, siteID, videoID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check site playback permission: %w", err)
	}
	return ok, nil
}

func (r *Repository) RandomApprovedVideos(ctx context.Context, siteID string, limit int, seedKey string) ([]Publication, error) {
	base := `
        SELECT ` + publicationColumns + `, ` + publicationVideoColumns + `
        FROM site_video_publications p
        INNER JOIN videos v ON v.id = p.video_id
        INNER JOIN categories c ON c.id = v.category_id
        WHERE p.deleted_at IS NULL
          AND p.site_id = $1
          AND p.status IN ('approved', 'synced')
          AND v.deleted_at IS NULL
          AND v.status = 'ready'
          AND v.visibility = 'public'
          AND c.deleted_at IS NULL
    `
	var query string
	args := []any{siteID}
	if seedKey != "" {
		query = base + ` ORDER BY md5(v.id::text || $2) LIMIT $3 `
		args = append(args, seedKey, limit)
	} else {
		query = base + ` ORDER BY random() LIMIT $2 `
		args = append(args, limit)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("random approved videos: %w", err)
	}
	defer rows.Close()

	items := []Publication{}
	for rows.Next() {
		p, err := scanPublicationWithVideo(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate random approved videos: %w", err)
	}
	return items, nil
}

func (r *Repository) ListCategories(ctx context.Context) ([]PublicationCategory, error) {
	const query = `
        SELECT id, name
        FROM categories
        WHERE deleted_at IS NULL AND status = 'active'
        ORDER BY sort_order ASC, name ASC
    `
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list sync categories: %w", err)
	}
	defer rows.Close()
	out := []PublicationCategory{}
	for rows.Next() {
		var c PublicationCategory
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, fmt.Errorf("scan sync category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPublication(row scanner) (*Publication, error) {
	var p Publication
	if err := row.Scan(
		&p.ID, &p.SiteID, &p.VideoID, &p.Status, &p.SyncMode, &p.RemotePostID, &p.RemoteSlug,
		&p.CategoryMapping, &p.Tags, &p.Notes, &p.LastSyncedAt,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPublicationWithVideo(row scanner) (*Publication, error) {
	p, err := scanPublicationPrefixWithVideo(row, nil)
	return p, err
}

func scanPublicationWithVideoAndTotal(row scanner) (*Publication, int, error) {
	var total int
	p, err := scanPublicationPrefixWithVideo(row, &total)
	return p, total, err
}

func scanPublicationPrefixWithVideo(row scanner, total *int) (*Publication, error) {
	var p Publication
	var v videos.Video
	dest := []any{
		&p.ID, &p.SiteID, &p.VideoID, &p.Status, &p.SyncMode, &p.RemotePostID, &p.RemoteSlug,
		&p.CategoryMapping, &p.Tags, &p.Notes, &p.LastSyncedAt,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		&v.ID, &v.CategoryID, &v.CategoryName,
		&v.Title, &v.Slug, &v.Description,
		&v.Status, &v.Visibility,
		&v.VideoFilename, &v.VideoOriginalName, &v.VideoMimeType, &v.VideoSizeBytes,
		&v.ThumbnailFilename, &v.ThumbnailSecond,
		&v.DurationSeconds, &v.ViewsCount, &v.UploadedByAdminID,
		&v.HLSStatus, &v.HLSReadyAt, &v.HLSError, &v.HLSVariants,
		&v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
	}
	if total != nil {
		dest = append(dest, total)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, fmt.Errorf("scan publication row: %w", err)
	}
	p.Video = &v
	return &p, nil
}

func scanDistributionRow(row scanner) (*DistributionRow, error) {
	var out DistributionRow
	var p Publication
	var pID, pSiteID, pVideoID, pStatus, pSyncMode, pRemotePostID, pRemoteSlug, pCategoryMapping, pNotes sql.NullString
	var pLastSyncedAt, pCreatedAt, pUpdatedAt, pDeletedAt sql.NullTime
	var pTags []string
	var remoteCategoryID, mappingSyncMode, remoteCategoryName sql.NullString
	var mappingEnabled, confirmedByWP, confirmedByStream sql.NullBool
	if err := row.Scan(
		&out.SiteID, &out.SiteName, &out.SiteDomain, &out.SiteStatus,
		&pID, &pSiteID, &pVideoID, &pStatus, &pSyncMode, &pRemotePostID, &pRemoteSlug,
		&pCategoryMapping, &pTags, &pNotes, &pLastSyncedAt,
		&pCreatedAt, &pUpdatedAt, &pDeletedAt,
		&remoteCategoryID, &mappingEnabled, &mappingSyncMode,
		&confirmedByWP, &confirmedByStream,
		&remoteCategoryName,
	); err != nil {
		return nil, fmt.Errorf("scan distribution row: %w", err)
	}
	if pID.Valid {
		p.ID = pID.String
		p.SiteID = pSiteID.String
		p.VideoID = pVideoID.String
		p.Status = pStatus.String
		if pSyncMode.Valid {
			p.SyncMode = &pSyncMode.String
		}
		if pRemotePostID.Valid {
			p.RemotePostID = &pRemotePostID.String
		}
		if pRemoteSlug.Valid {
			p.RemoteSlug = &pRemoteSlug.String
		}
		if pCategoryMapping.Valid {
			p.CategoryMapping = &pCategoryMapping.String
		}
		p.Tags = pTags
		if pNotes.Valid {
			p.Notes = &pNotes.String
		}
		if pLastSyncedAt.Valid {
			p.LastSyncedAt = &pLastSyncedAt.Time
		}
		p.CreatedAt = pCreatedAt.Time
		p.UpdatedAt = pUpdatedAt.Time
		if pDeletedAt.Valid {
			p.DeletedAt = &pDeletedAt.Time
		}
		out.Publication = &p
	}
	out.MappingStatus = categorymappings.MappingStatusUnmapped
	out.MappingSyncMode = categorymappings.SyncModeBlock
	if remoteCategoryID.Valid {
		out.RemoteCategoryID = &remoteCategoryID.String
	}
	if remoteCategoryName.Valid {
		out.RemoteCategoryName = &remoteCategoryName.String
	}
	if mappingSyncMode.Valid {
		out.MappingSyncMode = mappingSyncMode.String
		out.MappingEnabled = mappingEnabled.Bool
		if !mappingEnabled.Bool {
			out.MappingStatus = categorymappings.MappingStatusDisabled
		} else if !confirmedByWP.Bool || !confirmedByStream.Bool {
			out.MappingStatus = categorymappings.MappingStatusUnconfirmed
		} else if remoteCategoryID.Valid {
			out.MappingStatus = categorymappings.MappingStatusMapped
		}
	}
	return &out, nil
}

func cleanPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func cleanTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func orderClauseFor(sort string) string {
	switch sort {
	case "title":
		return "ORDER BY v.title ASC, p.updated_at DESC"
	case "-title":
		return "ORDER BY v.title DESC, p.updated_at DESC"
	case "updated_at":
		return "ORDER BY v.updated_at ASC"
	case "-updated_at":
		return "ORDER BY v.updated_at DESC"
	case "created_at":
		return "ORDER BY v.created_at ASC"
	case "-created_at", "":
		return "ORDER BY v.created_at DESC"
	default:
		return "ORDER BY p.updated_at DESC"
	}
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
