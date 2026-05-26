package publications

import (
	"time"

	"github.com/video-storage-api/internal/videos"
)

const (
	StatusDraft    = "draft"
	StatusApproved = "approved"
	StatusSynced   = "synced"
	StatusDisabled = "disabled"
	StatusFailed   = "failed"
	StatusBlocked  = "blocked"
)

type Publication struct {
	ID              string
	SiteID          string
	VideoID         string
	Status          string
	SyncMode        *string
	RemotePostID    *string
	RemoteSlug      *string
	CategoryMapping *string
	Tags            []string
	Notes           *string
	LastSyncedAt    *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time

	Video *videos.Video
}

func IsValidStatus(status string) bool {
	switch status {
	case StatusDraft, StatusApproved, StatusSynced, StatusDisabled, StatusFailed, StatusBlocked:
		return true
	default:
		return false
	}
}

func IsVisibleToSite(status string) bool {
	return status == StatusApproved || status == StatusSynced
}
