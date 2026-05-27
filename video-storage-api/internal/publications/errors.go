package publications

import "errors"

var (
	ErrNotFound                = errors.New("publication not found")
	ErrInvalidStatus           = errors.New("invalid publication status")
	ErrInvalidSyncMode         = errors.New("invalid publication sync mode")
	ErrInvalidVideoID          = errors.New("invalid video id")
	ErrInvalidUnmappedBehavior = errors.New("invalid unmapped behavior")
	ErrBlockedCategorySync     = errors.New("category mapping blocks publish sync")
	ErrMissingCategoryMapping  = errors.New("category mapping is missing or disabled")
)
