package alloweddomains

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const columns = `
    id, domain, type, status, notes, created_at, updated_at, deleted_at
`

type CreateInput struct {
	Domain string
	Type   string
	Status string
	Notes  *string
}

type UpdateInput struct {
	Domain *string
	Type   *string
	Status *string
	Notes  *string
}

type ListFilters struct {
	Search string
	Type   string
	Status string
	Limit  int
	Offset int
}

func (r *Repository) Insert(ctx context.Context, in CreateInput) (*Domain, error) {
	const query = `
        INSERT INTO allowed_domains (domain, type, status, notes)
        VALUES ($1, $2, $3, $4)
        RETURNING ` + columns
	row := r.pool.QueryRow(ctx, query, in.Domain, in.Type, in.Status, in.Notes)
	d, err := scan(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDomainTaken
		}
		return nil, fmt.Errorf("insert allowed domain: %w", err)
	}
	return d, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Domain, error) {
	const query = `
        SELECT ` + columns + `
        FROM allowed_domains
        WHERE id = $1 AND deleted_at IS NULL
        LIMIT 1
    `
	d, err := scan(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find allowed domain: %w", err)
	}
	return d, nil
}

func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*Domain, error) {
	const query = `
        UPDATE allowed_domains
        SET domain = COALESCE($2, domain),
            type = COALESCE($3, type),
            status = COALESCE($4, status),
            notes = COALESCE($5, notes),
            updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING ` + columns
	row := r.pool.QueryRow(ctx, query, id, in.Domain, in.Type, in.Status, in.Notes)
	d, err := scan(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrDomainTaken
		}
		return nil, fmt.Errorf("update allowed domain: %w", err)
	}
	return d, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const query = `
        UPDATE allowed_domains
        SET deleted_at = NOW(), updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete allowed domain: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, f ListFilters) ([]Domain, int, error) {
	const query = `
        SELECT ` + columns + `, COUNT(*) OVER() AS total_count
        FROM allowed_domains
        WHERE deleted_at IS NULL
          AND ($1::text IS NULL OR domain ILIKE $1 OR notes ILIKE $1)
          AND ($2::text IS NULL OR type = $2 OR ($2 = 'stream' AND type = 'both') OR ($2 = 'cors' AND type = 'both'))
          AND ($3::text IS NULL OR status = $3)
        ORDER BY created_at DESC
        LIMIT $4 OFFSET $5
    `
	var searchArg, typeArg, statusArg any
	if s := strings.TrimSpace(f.Search); s != "" {
		searchArg = "%" + s + "%"
	}
	if f.Type != "" {
		typeArg = f.Type
	}
	if f.Status != "" {
		statusArg = f.Status
	}
	rows, err := r.pool.Query(ctx, query, searchArg, typeArg, statusArg, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list allowed domains: %w", err)
	}
	defer rows.Close()

	items := []Domain{}
	total := 0
	for rows.Next() {
		var d Domain
		var rowTotal int
		if err := rows.Scan(&d.ID, &d.Domain, &d.Type, &d.Status, &d.Notes, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt, &rowTotal); err != nil {
			return nil, 0, fmt.Errorf("scan allowed domain: %w", err)
		}
		items = append(items, d)
		total = rowTotal
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate allowed domains: %w", err)
	}
	return items, total, nil
}

func (r *Repository) ActiveDomains(ctx context.Context, kind string) ([]string, error) {
	const query = `
        SELECT domain
        FROM allowed_domains
        WHERE deleted_at IS NULL
          AND status = 'active'
          AND (type = $1 OR type = 'both')
    `
	rows, err := r.pool.Query(ctx, query, kind)
	if err != nil {
		return nil, fmt.Errorf("list active allowed domains: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scan active allowed domain: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner) (*Domain, error) {
	var d Domain
	if err := row.Scan(&d.ID, &d.Domain, &d.Type, &d.Status, &d.Notes, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
