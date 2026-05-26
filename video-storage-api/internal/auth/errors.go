package auth

import "errors"

// Typed errors returned by the auth service. Handlers map these to HTTP
// status codes and safe public messages; the service never returns raw DB
// errors past this layer.
var (
	// ErrInvalidCredentials is returned when the email is unknown OR the
	// password does not match. Same error for both cases by design — we
	// must not leak which email addresses are valid.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAccountDisabled is returned when admin_users.status = 'disabled'.
	ErrAccountDisabled = errors.New("account disabled")

	// ErrAccountLocked is returned when the account is currently within a
	// lockout window (locked_until > now).
	ErrAccountLocked = errors.New("account temporarily locked")

	// ErrInvalidToken covers any token problem: missing, malformed,
	// expired, signature mismatch, or referencing a deleted user.
	ErrInvalidToken = errors.New("invalid token")
)
