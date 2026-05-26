package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrAdminUserNotFound is returned by lookups when no matching row exists.
var ErrAdminUserNotFound = errors.New("admin user not found")

// AdminUserRepository persists and retrieves AdminUser rows.
// No HTTP/business-rule logic lives here — only SQL.
type AdminUserRepository struct {
	pool *pgxpool.Pool
}

func NewAdminUserRepository(pool *pgxpool.Pool) *AdminUserRepository {
	return &AdminUserRepository{pool: pool}
}

const adminUserColumns = `
    id, email, password_hash, display_name, role, status,
    last_login_at, failed_login_attempts, locked_until,
    created_at, updated_at, deleted_at
`

// Create inserts a new admin user and returns the persisted row.
// Caller must pre-hash the password — this layer never sees plain text.
func (r *AdminUserRepository) Create(ctx context.Context, in NewAdminUserInput) (*AdminUser, error) {
	if in.PasswordHash == "" {
		return nil, errors.New("password hash is required")
	}
	if in.Email == "" {
		return nil, errors.New("email is required")
	}
	if in.DisplayName == "" {
		return nil, errors.New("display name is required")
	}

	role := in.Role
	if role == "" {
		role = RoleAdmin
	}
	status := in.Status
	if status == "" {
		status = StatusActive
	}

	const query = `
        INSERT INTO admin_users (email, password_hash, display_name, role, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING ` + adminUserColumns

	row := r.pool.QueryRow(ctx, query,
		strings.TrimSpace(in.Email),
		in.PasswordHash,
		in.DisplayName,
		role,
		status,
	)

	user, err := scanAdminUser(row)
	if err != nil {
		return nil, fmt.Errorf("insert admin user: %w", err)
	}
	return user, nil
}

// FindByEmail returns the non-deleted admin matching the email (case-insensitive).
func (r *AdminUserRepository) FindByEmail(ctx context.Context, email string) (*AdminUser, error) {
	const query = `
        SELECT ` + adminUserColumns + `
        FROM admin_users
        WHERE LOWER(email) = LOWER($1)
          AND deleted_at IS NULL
        LIMIT 1
    `

	row := r.pool.QueryRow(ctx, query, strings.TrimSpace(email))
	user, err := scanAdminUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminUserNotFound
		}
		return nil, fmt.Errorf("find admin user by email: %w", err)
	}
	return user, nil
}

// FindByID returns the non-deleted admin with the given UUID.
func (r *AdminUserRepository) FindByID(ctx context.Context, id string) (*AdminUser, error) {
	const query = `
        SELECT ` + adminUserColumns + `
        FROM admin_users
        WHERE id = $1
          AND deleted_at IS NULL
        LIMIT 1
    `

	row := r.pool.QueryRow(ctx, query, id)
	user, err := scanAdminUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminUserNotFound
		}
		return nil, fmt.Errorf("find admin user by id: %w", err)
	}
	return user, nil
}

// ExistsByEmail reports whether an active (non-deleted) admin uses this email.
func (r *AdminUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1 FROM admin_users
            WHERE LOWER(email) = LOWER($1)
              AND deleted_at IS NULL
        )
    `

	var exists bool
	if err := r.pool.QueryRow(ctx, query, strings.TrimSpace(email)).Scan(&exists); err != nil {
		return false, fmt.Errorf("check admin user existence: %w", err)
	}
	return exists, nil
}

// RecordFailedAttempt atomically increments failed_login_attempts and,
// when the new value reaches `threshold`, sets locked_until + status='locked'.
// The pre-computed `lockUntil` timestamp is applied only when the threshold
// is hit, so callers can pass it unconditionally.
// Returns the updated user row.
func (r *AdminUserRepository) RecordFailedAttempt(
	ctx context.Context,
	id string,
	threshold int,
	lockUntil time.Time,
) (*AdminUser, error) {
	const query = `
        UPDATE admin_users
        SET failed_login_attempts = failed_login_attempts + 1,
            locked_until = CASE
                WHEN failed_login_attempts + 1 >= $2 THEN $3::timestamptz
                ELSE locked_until
            END,
            status = CASE
                WHEN failed_login_attempts + 1 >= $2 THEN 'locked'
                ELSE status
            END,
            updated_at = NOW()
        WHERE id = $1
          AND deleted_at IS NULL
        RETURNING ` + adminUserColumns

	row := r.pool.QueryRow(ctx, query, id, threshold, lockUntil)
	user, err := scanAdminUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminUserNotFound
		}
		return nil, fmt.Errorf("record failed attempt: %w", err)
	}
	return user, nil
}

// MarkLoginSuccess resets login state after a successful authentication:
// zeroes failed_login_attempts, clears locked_until, stamps last_login_at,
// and re-activates a previously-locked status.
// Returns the updated user row.
func (r *AdminUserRepository) MarkLoginSuccess(ctx context.Context, id string) (*AdminUser, error) {
	const query = `
        UPDATE admin_users
        SET failed_login_attempts = 0,
            locked_until = NULL,
            last_login_at = NOW(),
            status = CASE WHEN status = 'locked' THEN 'active' ELSE status END,
            updated_at = NOW()
        WHERE id = $1
          AND deleted_at IS NULL
        RETURNING ` + adminUserColumns

	row := r.pool.QueryRow(ctx, query, id)
	user, err := scanAdminUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminUserNotFound
		}
		return nil, fmt.Errorf("mark login success: %w", err)
	}
	return user, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAdminUser(row scanner) (*AdminUser, error) {
	var u AdminUser
	if err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.Role,
		&u.Status,
		&u.LastLoginAt,
		&u.FailedLoginAttempts,
		&u.LockedUntil,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}
