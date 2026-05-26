package videos

import "errors"

var (
	ErrNotFound          = errors.New("video not found")
	ErrSlugTaken         = errors.New("video slug already exists")
	ErrInvalidSlug       = errors.New("invalid slug")
	ErrInvalidTitle      = errors.New("invalid title")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidVisibility = errors.New("invalid visibility")
	ErrCategoryNotFound  = errors.New("category not found")
	ErrInvalidSiteID     = errors.New("invalid site id")

	// ErrUnsupportedVideoType is returned when the uploaded video's sniffed
	// MIME isn't in the allow list.
	ErrUnsupportedVideoType = errors.New("unsupported video format")

	// ErrVideoTooLarge surfaces an upload size violation.
	ErrVideoTooLarge = errors.New("video file exceeds size limit")

	// ErrMissingVideo is returned when the multipart form is missing the
	// `video` file part.
	ErrMissingVideo = errors.New("video file is required")

	// ErrInvalidThumbnailSecond is returned when the requested second is
	// out of range (negative or above storage.MaxThumbnailSecond) OR not
	// parseable as an integer.
	ErrInvalidThumbnailSecond = errors.New("invalid thumbnail second")

	// ErrThumbnailGeneration covers any failure of the ffmpeg pipeline.
	// In Phase 12 the service clamps thumbnail_second against the known
	// duration and falls back to seconds 1 and 0, so this only fires
	// when the video file itself is unreadable.
	ErrThumbnailGeneration = errors.New("thumbnail generation failed; try a lower second or re-upload the video")
)
