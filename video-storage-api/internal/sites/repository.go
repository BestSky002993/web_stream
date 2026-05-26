package sites

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns SQL for `sites`.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const siteColumns = `
    id, name, domain, api_key_hash, status, daily_request_limit,
    created_at, updated_at, deleted_at
`

// CreateInput is the persistence-layer shape after validation.
type CreateInput struct {
	Name              string
	Domain            string
	APIKeyHash        string
	Status            string
	DailyRequestLimit int
}

// UpdateInput uses pointers — nil means "no change".
type UpdateInput struct {
	Name              *string
	Domain            *string
	Status            *string
	DailyRequestLimit *int
}

// ListFilters is the filter set for List.
type ListFilters struct {
	Search string // matched against name and domain
	Status string
	Limit  int
	Offset int
}

// Insert creates a new site row.
func (r *Repository) Insert(ctx context.Context, in CreateInput) (*Site, error) {
	const query = `
        INSERT INTO sites (name, domain, api_key_hash, status, daily_request_limit)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING ` + siteColumns

	row := r.pool.QueryRow(ctx, query,
		in.Name,
		strings.ToLower(strings.TrimSpace(in.Domain)),
		in.APIKeyHash,
		in.Status,
		in.DailyRequestLimit,
	)
	s, err := scanSite(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDomainTaken
		}
		return nil, fmt.Errorf("insert site: %w", err)
	}
	return s, nil
}

// FindByID returns the non-deleted site or ErrNotFound.
func (r *Repository) FindByID(ctx context.Context, id string) (*Site, error) {
	const query = `
        SELECT ` + siteColumns + `
        FROM sites
        WHERE id = $1 AND deleted_at IS NULL
        LIMIT 1
    `
	row := r.pool.QueryRow(ctx, query, id)
	s, err := scanSite(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find site by id: %w", err)
	}
	return s, nil
}

// FindByAPIKeyHash returns the non-deleted site matching the given
// SHA-256 hex hash. Used by the X-API-Key middleware on every request.
func (r *Repository) FindByAPIKeyHash(ctx context.Context, hash string) (*Site, error) {
	const query = `
        SELECT ` + siteColumns + `
        FROM sites
        WHERE api_key_hash = $1 AND deleted_at IS NULL
        LIMIT 1
    `
	row := r.pool.QueryRow(ctx, query, hash)
	s, err := scanSite(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find site by api key: %w", err)
	}
	return s, nil
}

// Update applies a partial update via COALESCE. Domain collisions
// surface as ErrDomainTaken; missing rows as ErrNotFound.
func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*Site, error) {
	// Normalise the domain to lower-case if provided. Pointer is left
	// pointing at the lower-cased value so COALESCE picks it up.
	var domainPtr *string
	if in.Domain != nil {
		d := strings.ToLower(strings.TrimSpace(*in.Domain))
		domainPtr = &d
	}

	const query = `
        UPDATE sites
        SET name                = COALESCE($2, name),
            domain              = COALESCE($3, domain),
            status              = COALESCE($4, status),
            daily_request_limit = COALESCE($5, daily_request_limit),
            updated_at          = NOW()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING ` + siteColumns

	row := r.pool.QueryRow(ctx, query,
		id,
		in.Name,
		domainPtr,
		in.Status,
		in.DailyRequestLimit,
	)
	s, err := scanSite(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrDomainTaken
		}
		return nil, fmt.Errorf("update site: %w", err)
	}
	return s, nil
}

// UpdateAPIKey replaces the stored hash and bumps updated_at.
func (r *Repository) UpdateAPIKey(ctx context.Context, id, newHash string) (*Site, error) {
	const query = `
        UPDATE sites
        SET api_key_hash = $2,
            updated_at   = NOW()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING ` + siteColumns

	row := r.pool.QueryRow(ctx, query, id, newHash)
	s, err := scanSite(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("rotate site api key: %w", err)
	}
	return s, nil
}

// SoftDelete sets deleted_at; the unique indices on domain and hash
// then free those values up for re-registration.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const query = `
        UPDATE sites
        SET deleted_at = NOW(),
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft delete site: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns a page of sites + total count via COUNT(*) OVER().
func (r *Repository) List(ctx context.Context, f ListFilters) ([]Site, int, error) {
	query := `
        SELECT ` + siteColumns + `, COUNT(*) OVER() AS total_count
        FROM sites
        WHERE deleted_at IS NULL
          AND ($1::text IS NULL OR (name ILIKE $1 OR domain ILIKE $1))
          AND ($2::text IS NULL OR status = $2)
        ORDER BY created_at DESC
        LIMIT $3 OFFSET $4
    `

	var searchArg, statusArg any
	if s := strings.TrimSpace(f.Search); s != "" {
		searchArg = "%" + s + "%"
	}
	if f.Status != "" {
		statusArg = f.Status
	}

	rows, err := r.pool.Query(ctx, query, searchArg, statusArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()

	items := []Site{}
	total := 0
	for rows.Next() {
		var s Site
		var rowTotal int
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Domain, &s.APIKeyHash, &s.Status, &s.DailyRequestLimit,
			&s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan site row: %w", err)
		}
		items = append(items, s)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate sites: %w", err)
	}
	return items, total, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSite(row scanner) (*Site, error) {
	var s Site
	if err := row.Scan(
		&s.ID, &s.Name, &s.Domain, &s.APIKeyHash, &s.Status, &s.DailyRequestLimit,
		&s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
