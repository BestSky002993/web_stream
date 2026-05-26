package streaming

import "errors"

var (
	// ErrNotFound is returned for unknown / soft-deleted videos.
	ErrNotFound = errors.New("video not found")

	// ErrNotAvailable is returned when the video exists but is not in a
	// state where streaming is allowed (status != ready).
	ErrNotAvailable = errors.New("video is not available for streaming")

	// ErrForbidden is returned for visibility / hotlink / domain
	// failures the client should not be able to recover from.
	ErrForbidden = errors.New("forbidden")

	// ErrSignatureRequired is returned when a private video is requested
	// without any signature parameters.
	ErrSignatureRequired = errors.New("signature required")

	// ErrSignatureExpired is returned when the `expires` field has
	// passed the current time.
	ErrSignatureExpired = errors.New("signature expired")

	// ErrSignatureInvalid is returned when the HMAC doesn't match or
	// the parameters are otherwise malformed.
	ErrSignatureInvalid = errors.New("signature invalid")

	// ErrInvalidRange is returned for unparseable / unsatisfiable
	// Range headers.
	ErrInvalidRange = errors.New("invalid range")

	// ErrFileMissing is returned when the DB row exists but the file on
	// disk does not — a server-side inconsistency.
	ErrFileMissing = errors.New("file missing on disk")
)
