package categorymappings

import "errors"

var (
	ErrNotFound      = errors.New("category mapping not found")
	ErrInvalidInput  = errors.New("invalid category mapping input")
	ErrInvalidSlug   = errors.New("invalid remote category slug")
	ErrInvalidMode   = errors.New("invalid sync mode")
	ErrCrossSiteData = errors.New("remote category does not belong to site")
)
