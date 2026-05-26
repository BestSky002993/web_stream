package categories

import "time"

// Status values accepted by categories.status (matches DB CHECK).
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// Category is the in-memory representation of a row in `categories`.
type Category struct {
	ID          string
	Name        string
	Slug        string
	Description *string
	Status      string
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
