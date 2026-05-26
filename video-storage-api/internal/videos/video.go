package videos

import "time"

// Status values accepted by videos.status (matches DB CHECK).
const (
	StatusDraft      = "draft"
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusBlocked    = "blocked"
	StatusDeleted    = "deleted"
)

// Visibility values accepted by videos.visibility (matches DB CHECK).
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// DefaultThumbnailSecond is used when a client omits the field on upload.
const DefaultThumbnailSecond = 1

// Video is the in-memory representation of a row in `videos` joined with
// the category's display name.
type Video struct {
	ID                string
	CategoryID        string
	CategoryName      string
	Title             string
	Slug              string
	Description       *string
	Status            string
	Visibility        string
	VideoFilename     string
	VideoOriginalName string
	VideoMimeType     string
	VideoSizeBytes    int64
	ThumbnailFilename string
	ThumbnailSecond   int
	DurationSeconds   *int
	ViewsCount        int64
	UploadedByAdminID string
	HLSStatus         string
	HLSReadyAt        *time.Time
	HLSError          *string
	HLSVariants       []string // nil before first successful conversion
	ApprovedSiteCount int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
