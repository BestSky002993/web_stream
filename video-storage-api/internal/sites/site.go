package sites

import "time"

// Status values accepted by sites.status (matches DB CHECK).
const (
	StatusActive    = "active"
	StatusDisabled  = "disabled"
	StatusSuspended = "suspended"
)

// DefaultDailyRequestLimit applies when the admin doesn't supply one.
const DefaultDailyRequestLimit = 10_000

// Site is the in-memory representation of a row in `sites`.
// APIKeyHash is the only key material stored — the plain key is never
// persisted and is returned to the admin exactly once.
type Site struct {
	ID                string
	Name              string
	Domain            string
	APIKeyHash        string
	Status            string
	DailyRequestLimit int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
