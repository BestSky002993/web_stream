package videos

import "time"

// VideoResponse is the canonical JSON projection of a Video as returned
// by the admin endpoints.
//
// `thumbnail_url` is the absolute public thumbnail endpoint — the same
// URL that public clients would use. Admins also get the raw filename
// inside `thumbnail.filename` for debugging.
type VideoResponse struct {
	ID                string               `json:"id"`
	Title             string               `json:"title"`
	Slug              string               `json:"slug"`
	Description       *string              `json:"description"`
	Status            string               `json:"status"`
	Visibility        string               `json:"visibility"`
	Category          VideoCategoryBrief   `json:"category"`
	Categories        []VideoCategoryBrief `json:"categories"`
	CategoryIDs       []string             `json:"category_ids"`
	Video             VideoFile            `json:"video"`
	Thumbnail         ThumbnailFile        `json:"thumbnail"`
	ThumbnailURL      string               `json:"thumbnail_url"`
	DurationSeconds   *int                 `json:"duration_seconds"`
	ViewsCount        int64                `json:"views_count"`
	UploadedByAdminID string               `json:"uploaded_by_admin_id"`
	HLS               HLSStatusResponse    `json:"hls"`
	ApprovedSiteCount int                  `json:"approved_site_count"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

// HLSStatusResponse mirrors the videos.hls_* columns so admins can
// see pipeline state alongside the rest of the metadata.
type HLSStatusResponse struct {
	Status   string     `json:"status"`   // pending / processing / ready / failed / skipped
	ReadyAt  *time.Time `json:"ready_at"` // set on first successful conversion
	Error    *string    `json:"error"`    // last failure message; nil on success
	Variants []string   `json:"variants"` // e.g. ["360p","480p","720p"]; nil before ready
}

// VideoCategoryBrief is the minimal category projection inside a video response.
type VideoCategoryBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VideoFile describes the stored video file (private storage — bare filename only).
type VideoFile struct {
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
}

// ThumbnailFile describes the auto-generated thumbnail. `second` is the
// timestamp the frame was captured at, in whole seconds.
type ThumbnailFile struct {
	Filename string `json:"filename"`
	Second   int    `json:"second"`
}

// ListVideosResponse is the body of GET /api/v1/admin/videos.
type ListVideosResponse struct {
	Data       []VideoResponse `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

// Pagination describes the slice of results being returned.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ErrorResponse is the canonical error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateVideoFormFields holds the multipart text fields for upload.
// The `video` file is accessed separately via c.FormFile. There is no
// `thumbnail` file part — thumbnails are extracted by ffmpeg.
//
// `ThumbnailSecond` is a string so we can distinguish "omitted" from a
// real 0; the handler parses and validates it.
type CreateVideoFormFields struct {
	Title           string `form:"title"            binding:"required,min=1,max=300"`
	Slug            string `form:"slug"             binding:"required,min=1,max=300"`
	Description     string `form:"description"      binding:"max=5000"`
	CategoryID      string `form:"category_id"      binding:"omitempty,uuid"`
	Visibility      string `form:"visibility"       binding:"omitempty,oneof=public private"`
	ThumbnailSecond string `form:"thumbnail_second"`
}

// UpdateVideoFormFields holds the optional multipart fields accepted by
// PUT /api/v1/admin/videos/:id. Empty string means "no change". Including
// `thumbnail_second` triggers a regeneration of the thumbnail at that
// second from the existing video file.
type UpdateVideoFormFields struct {
	Title           string `form:"title"            binding:"omitempty,min=1,max=300"`
	Slug            string `form:"slug"             binding:"omitempty,min=1,max=300"`
	Description     string `form:"description"      binding:"max=5000"`
	CategoryID      string `form:"category_id"      binding:"omitempty,uuid"`
	Status          string `form:"status"           binding:"omitempty,oneof=draft processing ready blocked"`
	Visibility      string `form:"visibility"       binding:"omitempty,oneof=public private"`
	ThumbnailSecond string `form:"thumbnail_second"`
}

func toResponse(v *Video, thumbnailURL string) VideoResponse {
	categories := make([]VideoCategoryBrief, 0, len(v.Categories))
	categoryIDs := make([]string, 0, len(v.Categories))
	for _, c := range v.Categories {
		categories = append(categories, VideoCategoryBrief{ID: c.ID, Name: c.Name})
		categoryIDs = append(categoryIDs, c.ID)
	}
	if len(categories) == 0 && v.CategoryID != "" {
		categories = append(categories, VideoCategoryBrief{ID: v.CategoryID, Name: v.CategoryName})
		categoryIDs = append(categoryIDs, v.CategoryID)
	}
	return VideoResponse{
		ID:          v.ID,
		Title:       v.Title,
		Slug:        v.Slug,
		Description: v.Description,
		Status:      v.Status,
		Visibility:  v.Visibility,
		Category: VideoCategoryBrief{
			ID:   v.CategoryID,
			Name: v.CategoryName,
		},
		Categories:  categories,
		CategoryIDs: categoryIDs,
		Video: VideoFile{
			Filename:     v.VideoFilename,
			OriginalName: v.VideoOriginalName,
			MimeType:     v.VideoMimeType,
			SizeBytes:    v.VideoSizeBytes,
		},
		Thumbnail: ThumbnailFile{
			Filename: v.ThumbnailFilename,
			Second:   v.ThumbnailSecond,
		},
		ThumbnailURL:      thumbnailURL,
		DurationSeconds:   v.DurationSeconds,
		ViewsCount:        v.ViewsCount,
		UploadedByAdminID: v.UploadedByAdminID,
		HLS: HLSStatusResponse{
			Status:   v.HLSStatus,
			ReadyAt:  v.HLSReadyAt,
			Error:    v.HLSError,
			Variants: v.HLSVariants,
		},
		ApprovedSiteCount: v.ApprovedSiteCount,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}
