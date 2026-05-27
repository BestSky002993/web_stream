package publications

import "time"

type UpsertPublicationRequest struct {
	Status          string     `json:"status" binding:"required,oneof=draft approved synced disabled failed blocked"`
	SyncMode        *string    `json:"sync_mode"`
	RemotePostID    *string    `json:"remote_post_id"`
	RemoteSlug      *string    `json:"remote_slug"`
	CategoryMapping *string    `json:"category_mapping"`
	Tags            []string   `json:"tags"`
	Notes           *string    `json:"notes"`
	LastSyncedAt    *time.Time `json:"last_synced_at"`
}

type PublicationResponse struct {
	ID              string            `json:"id"`
	SiteID          string            `json:"site_id"`
	VideoID         string            `json:"video_id"`
	Status          string            `json:"status"`
	SyncMode        *string           `json:"sync_mode"`
	RemotePostID    *string           `json:"remote_post_id"`
	RemoteSlug      *string           `json:"remote_slug"`
	CategoryMapping *string           `json:"category_mapping"`
	Tags            []string          `json:"tags"`
	Notes           *string           `json:"notes"`
	LastSyncedAt    *time.Time        `json:"last_synced_at"`
	Video           *PublicationVideo `json:"video,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type PublicationVideo struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	Slug            string                `json:"slug"`
	Description     *string               `json:"description"`
	Category        PublicationCategory   `json:"category"`
	Categories      []PublicationCategory `json:"categories"`
	CategoryIDs     []string              `json:"category_ids"`
	Status          string                `json:"status"`
	Visibility      string                `json:"visibility"`
	DurationSeconds *int                  `json:"duration_seconds"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type PublicationCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListPublicationsResponse struct {
	Data       []PublicationResponse `json:"data"`
	Pagination Pagination            `json:"pagination"`
}

type SiteDistributionRequest struct {
	SiteID   string `json:"site_id" binding:"required,uuid"`
	Status   string `json:"status" binding:"required,oneof=draft approved blocked"`
	SyncMode string `json:"sync_mode" binding:"required,oneof=publish draft block"`
}

type BulkDistributionRequest struct {
	VideoIDs         []string `json:"video_ids" binding:"required,min=1,max=1000"`
	Status           string   `json:"status" binding:"required,oneof=draft approved blocked"`
	SyncMode         string   `json:"sync_mode" binding:"required,oneof=publish draft block"`
	UnmappedBehavior string   `json:"unmapped_behavior" binding:"omitempty,oneof=skip draft"`
}

type BulkDistributionResponse struct {
	Approved int                       `json:"approved"`
	Updated  int                       `json:"updated"`
	Skipped  int                       `json:"skipped"`
	Errors   []BulkDistributionError   `json:"errors"`
	Warnings []BulkDistributionWarning `json:"warnings,omitempty"`
}

type BulkDistributionError struct {
	VideoID string `json:"video_id,omitempty"`
	Error   string `json:"error"`
}

type BulkDistributionWarning struct {
	VideoID string `json:"video_id"`
	Warning string `json:"warning"`
}

type SiteDistributionRow struct {
	SiteID             string                        `json:"site_id"`
	SiteName           string                        `json:"site_name"`
	SiteDomain         string                        `json:"site_domain"`
	SiteStatus         string                        `json:"site_status"`
	PublicationID      *string                       `json:"publication_id"`
	Status             string                        `json:"status"`
	SyncMode           string                        `json:"sync_mode"`
	EffectiveSyncMode  string                        `json:"effective_sync_mode"`
	MappingStatus      string                        `json:"mapping_status"`
	MappingSyncMode    string                        `json:"mapping_sync_mode"`
	MappingEnabled     bool                          `json:"mapping_enabled"`
	RemoteCategoryID   *string                       `json:"remote_category_id"`
	RemoteCategoryName *string                       `json:"remote_category_name"`
	CategoryMappings   []DistributionCategoryMapping `json:"category_mappings"`
	Warnings           []string                      `json:"warnings"`
	UpdatedAt          *time.Time                    `json:"updated_at"`
	Publication        *PublicationResponse          `json:"publication,omitempty"`
}

type SiteDistributionResponse struct {
	VideoID string                `json:"video_id"`
	Data    []SiteDistributionRow `json:"data"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func ToResponse(p *Publication) PublicationResponse {
	r := PublicationResponse{
		ID:              p.ID,
		SiteID:          p.SiteID,
		VideoID:         p.VideoID,
		Status:          p.Status,
		SyncMode:        p.SyncMode,
		RemotePostID:    p.RemotePostID,
		RemoteSlug:      p.RemoteSlug,
		CategoryMapping: p.CategoryMapping,
		Tags:            p.Tags,
		Notes:           p.Notes,
		LastSyncedAt:    p.LastSyncedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
	if p.Video != nil {
		categories := make([]PublicationCategory, 0, len(p.Video.Categories))
		categoryIDs := make([]string, 0, len(p.Video.Categories))
		for _, c := range p.Video.Categories {
			categories = append(categories, PublicationCategory{ID: c.ID, Name: c.Name})
			categoryIDs = append(categoryIDs, c.ID)
		}
		if len(categories) == 0 && p.Video.CategoryID != "" {
			categories = append(categories, PublicationCategory{ID: p.Video.CategoryID, Name: p.Video.CategoryName})
			categoryIDs = append(categoryIDs, p.Video.CategoryID)
		}
		r.Video = &PublicationVideo{
			ID:          p.Video.ID,
			Title:       p.Video.Title,
			Slug:        p.Video.Slug,
			Description: p.Video.Description,
			Category: PublicationCategory{
				ID:   p.Video.CategoryID,
				Name: p.Video.CategoryName,
			},
			Categories:      categories,
			CategoryIDs:     categoryIDs,
			Status:          p.Video.Status,
			Visibility:      p.Video.Visibility,
			DurationSeconds: p.Video.DurationSeconds,
			UpdatedAt:       p.Video.UpdatedAt,
		}
	}
	return r
}

func ToResponseList(items []Publication) []PublicationResponse {
	out := make([]PublicationResponse, len(items))
	for i := range items {
		out[i] = ToResponse(&items[i])
	}
	return out
}
