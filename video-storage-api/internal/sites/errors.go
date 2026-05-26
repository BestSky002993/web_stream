package sites

import "errors"

var (
	ErrNotFound        = errors.New("site not found")
	ErrDomainTaken     = errors.New("domain already in use")
	ErrInvalidName     = errors.New("invalid name")
	ErrInvalidDomain   = errors.New("invalid domain")
	ErrInvalidStatus   = errors.New("invalid status")
	ErrInvalidLimit    = errors.New("invalid daily_request_limit")

	// Returned by the site-auth middleware.
	ErrMissingAPIKey   = errors.New("api key required")
	ErrInvalidAPIKey   = errors.New("invalid api key")
	ErrSiteDisabled    = errors.New("site is disabled")
	ErrSiteSuspended   = errors.New("site is suspended")
	ErrDomainMismatch  = errors.New("request domain not allowed for this site")
)
