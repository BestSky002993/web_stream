package categories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns SQL for `categories`. It has no knowledge of HTTP
// or business validation — those live in the service.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const categoryColumns = `
    id, name, slug, description, status, sort_order,
    created_at, updated_at, deleted_at
`

const categoryListColumns = `
    id, name, slug, description, status, sort_order,
    (
        SELECT COUNT(*)
        FROM video_category_assignments vca
        INNER JOIN videos v ON v.id = vca.video_id
        WHERE vca.category_id = categories.id
          AND v.deleted_at IS NULL
    ) AS video_count,
    created_at, updated_at, deleted_at
`

// CreateInput is the persistence-layer shape (already validated/normalized).
type CreateInput struct {
	Name        string
	Slug        string
	Description *string
	Status      string
	SortOrder   int
}

// UpdateInput uses pointers so a nil field means "don't change".
type UpdateInput struct {
	Name        *string
	Slug        *string
	Description *string
	Status      *string
	SortOrder   *int
}

// ListFilters is the filter set accepted by List.
type ListFilters struct {
	Search string // matched against name and slug, ILIKE; empty = no filter
	Status string // empty = any status
	Sort   string // whitelisted: "", sort_order, -sort_order, name, -name, created_at, -created_at
	Limit  int
	Offset int
}

// Insert creates a new row and returns it. A duplicate slug surfaces as
// ErrSlugTaken so callers can map it to 409.
func (r *Repository) Insert(ctx context.Context, in CreateInput) (*Category, error) {
	const query = `
        INSERT INTO categories (name, slug, description, status, sort_order)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING ` + categoryColumns

	row := r.pool.QueryRow(ctx, query,
		in.Name, in.Slug, in.Description, in.Status, in.SortOrder,
	)

	c, err := scanCategory(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("insert category: %w", err)
	}
	return c, nil
}

// FindByID returns the non-deleted row matching id or ErrNotFound.
func (r *Repository) FindByID(ctx context.Context, id string) (*Category, error) {
	const query = `
        SELECT ` + categoryColumns + `
        FROM categories
        WHERE id = $1 AND deleted_at IS NULL
        LIMIT 1
    `
	row := r.pool.QueryRow(ctx, query, id)
	c, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find category by id: %w", err)
	}
	return c, nil
}

// Update applies a partial update. Nil pointer fields are left unchanged.
// A duplicate slug surfaces as ErrSlugTaken; a missing/deleted row as ErrNotFound.
func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*Category, error) {
	const query = `
        UPDATE categories
        SET name        = COALESCE($2, name),
            slug        = COALESCE($3, slug),
            description = COALESCE($4, description),
            status      = COALESCE($5, status),
            sort_order  = COALESCE($6, sort_order),
            updated_at  = NOW()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING ` + categoryColumns

	row := r.pool.QueryRow(ctx, query,
		id,
		in.Name,
		in.Slug,
		in.Description,
		in.Status,
		in.SortOrder,
	)

	c, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("update category: %w", err)
	}
	return c, nil
}

// SoftDelete sets deleted_at = NOW() if the row exists and is live.
// Returns ErrNotFound otherwise.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const query = `
        UPDATE categories
        SET deleted_at = NOW(),
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns a page of categories plus the total count of rows that
// match the filters (ignoring limit/offset). Uses a COUNT(*) OVER()
// window so data + total are consistent within a single query.
func (r *Repository) List(ctx context.Context, f ListFilters) ([]Category, int, error) {
	orderClause := orderClauseFor(f.Sort)

	// $1=search (text|nil) $2=status (text|nil) $3=limit $4=offset
	query := `
        SELECT ` + categoryListColumns + `, COUNT(*) OVER() AS total_count
        FROM categories
        WHERE deleted_at IS NULL
          AND ($1::text IS NULL OR (name ILIKE $1 OR slug ILIKE $1))
          AND ($2::text IS NULL OR status = $2)
        ` + orderClause + `
        LIMIT $3 OFFSET $4
    `

	var searchArg any
	if f.Search != "" {
		searchArg = "%" + f.Search + "%"
	}
	var statusArg any
	if f.Status != "" {
		statusArg = f.Status
	}

	rows, err := r.pool.Query(ctx, query, searchArg, statusArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var (
		items []Category
		total int
	)
	for rows.Next() {
		var c Category
		var rowTotal int
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Slug, &c.Description, &c.Status, &c.SortOrder,
			&c.VideoCount,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan category row: %w", err)
		}
		items = append(items, c)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate categories: %w", err)
	}

	// If there were no matching rows the window function never fires;
	// fall through with total=0 and an empty slice.
	if items == nil {
		items = []Category{}
	}

	return items, total, nil
}

// orderClauseFor returns a static, whitelisted ORDER BY clause.
// All branches are hard-coded strings — never interpolate user input.
func orderClauseFor(sort string) string {
	switch sort {
	case "name":
		return "ORDER BY name ASC, created_at DESC"
	case "-name":
		return "ORDER BY name DESC, created_at DESC"
	case "created_at":
		return "ORDER BY created_at ASC"
	case "-created_at":
		return "ORDER BY created_at DESC"
	case "sort_order":
		return "ORDER BY sort_order ASC, created_at DESC"
	case "-sort_order":
		return "ORDER BY sort_order DESC, created_at DESC"
	default:
		// default sort
		return "ORDER BY sort_order ASC, created_at DESC"
	}
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCategory(row scanner) (*Category, error) {
	var c Category
	if err := row.Scan(
		&c.ID, &c.Name, &c.Slug, &c.Description, &c.Status, &c.SortOrder,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
