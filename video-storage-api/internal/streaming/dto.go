package streaming

import "time"

// PublicVideoResponse is the canonical JSON shape returned by the
// public list / detail / random endpoints.
//
// It deliberately omits anything that could leak server internals:
// raw storage filenames, the uploader's ID, the on-disk MIME type,
// the upload size, the random ordering seed, etc.
type PublicVideoResponse struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	Slug            string                `json:"slug"`
	Description     *string               `json:"description"`
	Category        PublicCategoryBrief   `json:"category"`
	Categories      []PublicCategoryBrief `json:"categories"`
	CategoryIDs     []string              `json:"category_ids"`
	ThumbnailURL    string                `json:"thumbnail_url"`
	DurationSeconds *int                  `json:"duration_seconds"`
	ViewsCount      int64                 `json:"views_count"`
	CreatedAt       time.Time             `json:"created_at"`
	StreamAvailable bool                  `json:"stream_available"`
}

// PublicCategoryBrief is the minimal category projection inside a public response.
// (Slug intentionally omitted — the Video model only carries the joined
// display name today; slug can be added in a later phase without breaking
// the response.)
type PublicCategoryBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PublicListResponse is the body of GET /api/v1/videos.
type PublicListResponse struct {
	Data       []PublicVideoResponse `json:"data"`
	Pagination PublicPagination      `json:"pagination"`
}

// PublicPagination describes the slice of results being returned.
type PublicPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// RandomListResponse is the body of GET /api/v1/videos/random.
// No `pagination` field — random feeds aren't pageable; clients ask
// for `limit` items at a time.
type RandomListResponse struct {
	Data []PublicVideoResponse `json:"data"`
	Seed string                `json:"seed,omitempty"`
}

type SyncRemoteCategory struct {
	ID             string `json:"id"`
	RemoteTermID   string `json:"remote_term_id"`
	RemoteSlug     string `json:"remote_slug"`
	RemoteName     string `json:"remote_name"`
	RemoteTaxonomy string `json:"remote_taxonomy"`
}

type SyncVideoResponse struct {
	VideoID               string                `json:"video_id"`
	Title                 string                `json:"title"`
	Slug                  string                `json:"slug"`
	Description           *string               `json:"description"`
	ThumbnailURL          string                `json:"thumbnail_url"`
	Category              PublicCategoryBrief   `json:"category"`
	Categories            []PublicCategoryBrief `json:"categories"`
	CategoryIDs           []string              `json:"category_ids"`
	BackendCategoryID     string                `json:"backend_category_id"`
	BackendCategoryName   string                `json:"backend_category_name"`
	CategoryMappingStatus string                `json:"category_mapping_status"`
	MappedRemoteCategory  *SyncRemoteCategory   `json:"mapped_remote_category"`
	SyncMode              string                `json:"sync_mode"`
	CategoryMapping       *string               `json:"category_mapping"`
	Tags                  []string              `json:"tags"`
	Actors                []string              `json:"actors"`
	DurationSeconds       *int                  `json:"duration_seconds"`
	UpdatedAt             time.Time             `json:"updated_at"`
	PublicationStatus     string                `json:"publication_status"`
	RemotePostID          *string               `json:"remote_post_id"`
	RemoteSlug            *string               `json:"remote_slug"`
}

type SyncVideoListResponse struct {
	Data       []SyncVideoResponse `json:"data"`
	Pagination PublicPagination    `json:"pagination"`
}

type SyncCategoryListResponse struct {
	Data []PublicCategoryBrief `json:"data"`
}

type SyncStatusRequest struct {
	Status       string  `json:"status" binding:"required,oneof=synced failed"`
	RemotePostID *string `json:"remote_post_id"`
	RemoteSlug   *string `json:"remote_slug"`
}
