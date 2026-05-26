package categories

import "errors"

// Typed errors returned by the service. Handlers translate these into
// HTTP status codes; the service never returns raw DB errors past itself.
var (
	// ErrNotFound is returned when a category id doesn't match a live row.
	ErrNotFound = errors.New("category not found")

	// ErrSlugTaken is returned when the requested slug collides with an
	// existing non-deleted row.
	ErrSlugTaken = errors.New("category slug already exists")

	// ErrInvalidSlug is returned when the provided slug is empty after
	// normalization (i.e. had no usable characters).
	ErrInvalidSlug = errors.New("invalid slug")

	// ErrInvalidName is returned when name is blank after trimming.
	ErrInvalidName = errors.New("invalid name")

	// ErrInvalidStatus is returned when status is set to an unsupported value.
	ErrInvalidStatus = errors.New("invalid status")
)
