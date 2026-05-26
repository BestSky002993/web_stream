package categorymappings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const remoteCategoryColumns = `
    id, site_id, remote_term_id, remote_slug, remote_name, remote_taxonomy,
    parent_remote_term_id, created_at, updated_at
`

func (r *Repository) ListRemoteCategories(ctx context.Context, siteID string) ([]RemoteCategory, error) {
	const query = `
        SELECT ` + remoteCategoryColumns + `
        FROM site_remote_categories
        WHERE site_id = $1
        ORDER BY remote_taxonomy ASC, remote_name ASC, remote_slug ASC
    `
	rows, err := r.pool.Query(ctx, query, siteID)
	if err != nil {
		return nil, fmt.Errorf("list remote categories: %w", err)
	}
	defer rows.Close()

	out := []RemoteCategory{}
	for rows.Next() {
		c, err := scanRemoteCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) ImportRemoteCategories(ctx context.Context, siteID string, items []RemoteCategoryImport) ([]RemoteCategory, error) {
	if len(items) == 0 {
		return r.ListRemoteCategories(ctx, siteID)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin remote category import: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
        INSERT INTO site_remote_categories (
            site_id, remote_term_id, remote_slug, remote_name, remote_taxonomy, parent_remote_term_id
        )
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (site_id, remote_taxonomy, remote_term_id)
        DO UPDATE SET
            remote_slug = EXCLUDED.remote_slug,
            remote_name = EXCLUDED.remote_name,
            parent_remote_term_id = EXCLUDED.parent_remote_term_id,
            updated_at = NOW()
    `
	for _, item := range items {
		if _, err := tx.Exec(ctx, query, siteID, item.RemoteTermID, item.RemoteSlug, item.RemoteName, item.RemoteTaxonomy, item.ParentRemoteTermID); err != nil {
			return nil, fmt.Errorf("import remote category: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit remote category import: %w", err)
	}
	return r.ListRemoteCategories(ctx, siteID)
}

func (r *Repository) ListBackendCategories(ctx context.Context) ([]BackendCategory, error) {
	const query = `
        SELECT id, name, slug
        FROM categories
        WHERE deleted_at IS NULL AND status = 'active'
        ORDER BY sort_order ASC, name ASC
    `
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list backend categories: %w", err)
	}
	defer rows.Close()
	out := []BackendCategory{}
	for rows.Next() {
		var c BackendCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil {
			return nil, fmt.Errorf("scan backend category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) ListMappings(ctx context.Context, siteID string) ([]MappingRow, error) {
	const query = `
        SELECT
            c.id, c.name, c.slug,
            m.id, m.site_id, m.backend_category_id, m.remote_category_id,
            m.enabled, m.sync_mode, m.created_by, m.confirmed_by_wp, m.confirmed_by_stream,
            m.created_at, m.updated_at,
            rc.id, rc.site_id, rc.remote_term_id, rc.remote_slug, rc.remote_name, rc.remote_taxonomy,
            rc.parent_remote_term_id, rc.created_at, rc.updated_at
        FROM categories c
        LEFT JOIN site_category_mappings m
            ON m.site_id = $1 AND m.backend_category_id = c.id
        LEFT JOIN site_remote_categories rc
            ON rc.id = m.remote_category_id AND rc.site_id = $1
        WHERE c.deleted_at IS NULL AND c.status = 'active'
        ORDER BY c.sort_order ASC, c.name ASC
    `
	rows, err := r.pool.Query(ctx, query, siteID)
	if err != nil {
		return nil, fmt.Errorf("list category mappings: %w", err)
	}
	defer rows.Close()
	out := []MappingRow{}
	for rows.Next() {
		row, err := scanMappingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertAdminMappings(ctx context.Context, siteID string, items []MappingUpsertInput) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin category mapping upsert: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
        INSERT INTO site_category_mappings (
            site_id, backend_category_id, remote_category_id, enabled, sync_mode,
            created_by, confirmed_by_stream
        )
        SELECT $1, $2, $3, $4, $5, 'stream', $6
        WHERE $3::uuid IS NULL OR EXISTS (
            SELECT 1 FROM site_remote_categories WHERE id = $3 AND site_id = $1
        )
        ON CONFLICT (site_id, backend_category_id)
        DO UPDATE SET
            remote_category_id = EXCLUDED.remote_category_id,
            enabled = EXCLUDED.enabled,
            sync_mode = EXCLUDED.sync_mode,
            confirmed_by_stream = EXCLUDED.confirmed_by_stream,
            updated_at = NOW()
    `
	for _, item := range items {
		tag, err := tx.Exec(ctx, query, siteID, item.BackendCategoryID, item.RemoteCategoryID, item.Enabled, item.SyncMode, item.ConfirmedByStream)
		if err != nil {
			return fmt.Errorf("upsert category mapping: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrCrossSiteData
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit category mapping upsert: %w", err)
	}
	return nil
}

func (r *Repository) UpdateSiteMapping(ctx context.Context, siteID, id string, in SiteMappingUpdateInput) (*MappingRow, error) {
	const query = `
        WITH resolved_remote AS (
            SELECT rc.id
            FROM site_remote_categories rc
            WHERE rc.site_id = $1
              AND (
                  rc.id::text = $3::text
                  OR rc.remote_term_id = $3::text
              )
            ORDER BY
              CASE
                WHEN rc.id::text = $3::text THEN 0
                WHEN rc.remote_taxonomy = 'category' THEN 1
                ELSE 2
              END,
              rc.updated_at DESC
            LIMIT 1
        ), confirmation AS (
            SELECT (SELECT id FROM resolved_remote) IS NOT NULL AS remote_exists
        ), updated AS (
            UPDATE site_category_mappings m
            SET remote_category_id = (SELECT id FROM resolved_remote),
                enabled = $4,
                sync_mode = $5,
                confirmed_by_wp = CASE WHEN (SELECT remote_exists FROM confirmation) THEN TRUE ELSE $6 END,
                confirmed_by_stream = CASE WHEN (SELECT remote_exists FROM confirmation) THEN TRUE ELSE FALSE END,
                updated_at = NOW()
            WHERE m.site_id = $1
              AND (m.id::text = $2 OR m.backend_category_id::text = $2)
              AND ($3::text IS NULL OR EXISTS (SELECT 1 FROM resolved_remote))
            RETURNING m.backend_category_id
        ), inserted AS (
            INSERT INTO site_category_mappings (
                site_id, backend_category_id, remote_category_id, enabled, sync_mode,
                created_by, confirmed_by_wp, confirmed_by_stream
            )
            SELECT
                $1, c.id, (SELECT id FROM resolved_remote), $4, $5, 'wp',
                CASE WHEN (SELECT remote_exists FROM confirmation) THEN TRUE ELSE $6 END,
                CASE WHEN (SELECT remote_exists FROM confirmation) THEN TRUE ELSE FALSE END
            FROM categories c
            WHERE NOT EXISTS (SELECT 1 FROM updated)
              AND c.id::text = $2
              AND c.deleted_at IS NULL
              AND c.status = 'active'
              AND ($3::text IS NULL OR EXISTS (SELECT 1 FROM resolved_remote))
            ON CONFLICT (site_id, backend_category_id)
            DO UPDATE SET
                remote_category_id = EXCLUDED.remote_category_id,
                enabled = EXCLUDED.enabled,
                sync_mode = EXCLUDED.sync_mode,
                confirmed_by_wp = EXCLUDED.confirmed_by_wp,
                confirmed_by_stream = EXCLUDED.confirmed_by_stream,
                updated_at = NOW()
            RETURNING backend_category_id
        )
        SELECT backend_category_id FROM updated
        UNION ALL
        SELECT backend_category_id FROM inserted
        LIMIT 1
    `
	var backendID string
	if err := r.pool.QueryRow(ctx, query, siteID, id, in.RemoteCategoryID, in.Enabled, in.SyncMode, in.ConfirmedByWP).Scan(&backendID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update site category mapping: %w", err)
	}
	rows, err := r.MappingStatusForCategories(ctx, siteID, []string{backendID})
	if err != nil {
		return nil, err
	}
	row, ok := rows[backendID]
	if !ok {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *Repository) SuggestSameNameMappings(ctx context.Context, siteID string) error {
	const query = `
        INSERT INTO site_category_mappings (
            site_id, backend_category_id, remote_category_id, enabled, sync_mode,
            created_by, confirmed_by_wp, confirmed_by_stream
        )
        SELECT DISTINCT ON (c.id)
            $1, c.id, rc.id, FALSE, 'block', 'suggestion', FALSE, FALSE
        FROM categories c
        INNER JOIN site_remote_categories rc ON rc.site_id = $1
        WHERE c.deleted_at IS NULL
          AND c.status = 'active'
          AND lower(trim(c.name)) = lower(trim(rc.remote_name))
          AND NOT EXISTS (
              SELECT 1 FROM site_category_mappings m
              WHERE m.site_id = $1 AND m.backend_category_id = c.id
          )
        ORDER BY c.id, rc.updated_at DESC
        ON CONFLICT (site_id, backend_category_id) DO NOTHING
    `
	if _, err := r.pool.Exec(ctx, query, siteID); err != nil {
		return fmt.Errorf("suggest same-name category mappings: %w", err)
	}
	return nil
}

func (r *Repository) MappingStatusForCategories(ctx context.Context, siteID string, categoryIDs []string) (map[string]MappingRow, error) {
	out := map[string]MappingRow{}
	if len(categoryIDs) == 0 {
		return out, nil
	}
	const query = `
        SELECT
            c.id, c.name, c.slug,
            m.id, m.site_id, m.backend_category_id, m.remote_category_id,
            m.enabled, m.sync_mode, m.created_by, m.confirmed_by_wp, m.confirmed_by_stream,
            m.created_at, m.updated_at,
            rc.id, rc.site_id, rc.remote_term_id, rc.remote_slug, rc.remote_name, rc.remote_taxonomy,
            rc.parent_remote_term_id, rc.created_at, rc.updated_at
        FROM categories c
        LEFT JOIN site_category_mappings m
            ON m.site_id = $1 AND m.backend_category_id = c.id
        LEFT JOIN site_remote_categories rc
            ON rc.id = m.remote_category_id AND rc.site_id = $1
        WHERE c.id::text = ANY($2::text[])
    `
	rows, err := r.pool.Query(ctx, query, siteID, categoryIDs)
	if err != nil {
		return nil, fmt.Errorf("lookup category mapping statuses: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		row, err := scanMappingRow(rows)
		if err != nil {
			return nil, err
		}
		out[row.BackendCategory.ID] = *row
	}
	return out, rows.Err()
}

func scanRemoteCategory(row interface{ Scan(dest ...any) error }) (*RemoteCategory, error) {
	var c RemoteCategory
	if err := row.Scan(
		&c.ID, &c.SiteID, &c.RemoteTermID, &c.RemoteSlug, &c.RemoteName, &c.RemoteTaxonomy,
		&c.ParentRemoteTermID, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan remote category: %w", err)
	}
	return &c, nil
}

func scanMappingRow(row interface{ Scan(dest ...any) error }) (*MappingRow, error) {
	var out MappingRow
	var mID, mSiteID, mBackendID, mRemoteID, mSyncMode, mCreatedBy sql.NullString
	var mEnabled, mWP, mStream sql.NullBool
	var mCreatedAt, mUpdatedAt sql.NullTime
	var rID, rSiteID, rTermID, rSlug, rName, rTaxonomy, rParent sql.NullString
	var rCreatedAt, rUpdatedAt sql.NullTime
	if err := row.Scan(
		&out.BackendCategory.ID, &out.BackendCategory.Name, &out.BackendCategory.Slug,
		&mID, &mSiteID, &mBackendID, &mRemoteID,
		&mEnabled, &mSyncMode, &mCreatedBy, &mWP, &mStream,
		&mCreatedAt, &mUpdatedAt,
		&rID, &rSiteID, &rTermID, &rSlug, &rName, &rTaxonomy,
		&rParent, &rCreatedAt, &rUpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan category mapping row: %w", err)
	}
	if mID.Valid {
		m := &CategoryMapping{
			ID:                mID.String,
			SiteID:            mSiteID.String,
			BackendCategoryID: mBackendID.String,
			Enabled:           mEnabled.Bool,
			SyncMode:          mSyncMode.String,
			ConfirmedByWP:     mWP.Bool,
			ConfirmedByStream: mStream.Bool,
			CreatedAt:         mCreatedAt.Time,
			UpdatedAt:         mUpdatedAt.Time,
		}
		if mRemoteID.Valid {
			m.RemoteCategoryID = &mRemoteID.String
		}
		if mCreatedBy.Valid {
			m.CreatedBy = &mCreatedBy.String
		}
		out.Mapping = m
	}
	if rID.Valid {
		r := &RemoteCategory{
			ID:             rID.String,
			SiteID:         rSiteID.String,
			RemoteTermID:   rTermID.String,
			RemoteSlug:     rSlug.String,
			RemoteName:     rName.String,
			RemoteTaxonomy: rTaxonomy.String,
			CreatedAt:      rCreatedAt.Time,
			UpdatedAt:      rUpdatedAt.Time,
		}
		if rParent.Valid {
			r.ParentRemoteTermID = &rParent.String
		}
		out.RemoteCategory = r
	}
	out.Status = MappingStatus(out.Mapping)
	return &out, nil
}
