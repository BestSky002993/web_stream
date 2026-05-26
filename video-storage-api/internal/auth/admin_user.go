package auth

import "time"

// Role values accepted by the admin_users.role column (matches DB CHECK).
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleEditor     = "editor"
)

// Status values accepted by the admin_users.status column (matches DB CHECK).
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusLocked   = "locked"
)

// AdminUser is the in-memory representation of a row in admin_users.
// PasswordHash holds the bcrypt hash — plain passwords never live here.
type AdminUser struct {
	ID                  string
	Email               string
	PasswordHash        string
	DisplayName         string
	Role                string
	Status              string
	LastLoginAt         *time.Time
	FailedLoginAttempts int
	LockedUntil         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}

// NewAdminUserInput captures everything needed to create a new admin row.
// The repository fills in defaults for any zero-valued fields it owns
// (id, timestamps, failed_login_attempts).
type NewAdminUserInput struct {
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string // optional; defaults to RoleAdmin
	Status       string // optional; defaults to StatusActive
}
