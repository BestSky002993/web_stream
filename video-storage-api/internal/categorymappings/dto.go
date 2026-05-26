package categorymappings

import "time"

type RemoteCategoryImportRequest struct {
	Categories []RemoteCategoryImportItem `json:"categories"`
	Data       []RemoteCategoryImportItem `json:"data"`
}

type RemoteCategoryImportItem struct {
	RemoteTermID       string  `json:"remote_term_id"`
	RemoteSlug         string  `json:"remote_slug"`
	RemoteName         string  `json:"remote_name"`
	RemoteTaxonomy     string  `json:"remote_taxonomy"`
	ParentRemoteTermID *string `json:"parent_remote_term_id"`
}

type RemoteCategoryListResponse struct {
	Data []RemoteCategoryResponse `json:"data"`
}

type RemoteCategoryResponse struct {
	ID                 string    `json:"id"`
	SiteID             string    `json:"site_id"`
	RemoteTermID       string    `json:"remote_term_id"`
	RemoteSlug         string    `json:"remote_slug"`
	RemoteName         string    `json:"remote_name"`
	RemoteTaxonomy     string    `json:"remote_taxonomy"`
	ParentRemoteTermID *string   `json:"parent_remote_term_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type MappingListResponse struct {
	Data []MappingResponse `json:"data"`
}

type MappingUpsertRequest struct {
	Mappings []MappingUpsertItem `json:"mappings" binding:"required"`
}

type MappingUpsertItem struct {
	BackendCategoryID string  `json:"backend_category_id" binding:"required"`
	RemoteCategoryID  *string `json:"remote_category_id"`
	Enabled           bool    `json:"enabled"`
	SyncMode          string  `json:"sync_mode" binding:"required,oneof=publish draft block"`
	ConfirmedByStream bool    `json:"confirmed_by_stream"`
}

type SiteMappingUpdateRequest struct {
	BackendCategoryID string  `json:"backend_category_id"`
	RemoteCategoryID  *string `json:"remote_category_id"`
	Enabled           bool    `json:"enabled"`
	SyncMode          string  `json:"sync_mode" binding:"required,oneof=publish draft block"`
	ConfirmedByWP     bool    `json:"confirmed_by_wp"`
}

type BackendCategoryResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type MappingResponse struct {
	BackendCategory   BackendCategoryResponse `json:"backend_category"`
	MappingID         *string                 `json:"mapping_id"`
	RemoteCategoryID  *string                 `json:"remote_category_id"`
	RemoteCategory    *RemoteCategoryResponse `json:"remote_category"`
	Enabled           bool                    `json:"enabled"`
	SyncMode          string                  `json:"sync_mode"`
	CreatedBy         *string                 `json:"created_by"`
	ConfirmedByWP     bool                    `json:"confirmed_by_wp"`
	ConfirmedByStream bool                    `json:"confirmed_by_stream"`
	WPConfirmed       bool                    `json:"wp_confirmed"`
	StreamConfirmed   bool                    `json:"stream_confirmed"`
	MappingConfirmed  bool                    `json:"mapping_confirmed"`
	Status            string                  `json:"status"`
	CreatedAt         *time.Time              `json:"created_at"`
	UpdatedAt         *time.Time              `json:"updated_at"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (r RemoteCategoryImportRequest) items() []RemoteCategoryImportItem {
	if r.Categories != nil {
		return r.Categories
	}
	return r.Data
}

func importItems(items []RemoteCategoryImportItem) []RemoteCategoryImport {
	out := make([]RemoteCategoryImport, len(items))
	for i := range items {
		out[i] = RemoteCategoryImport{
			RemoteTermID:       items[i].RemoteTermID,
			RemoteSlug:         items[i].RemoteSlug,
			RemoteName:         items[i].RemoteName,
			RemoteTaxonomy:     items[i].RemoteTaxonomy,
			ParentRemoteTermID: items[i].ParentRemoteTermID,
		}
	}
	return out
}

func upsertItems(items []MappingUpsertItem) []MappingUpsertInput {
	out := make([]MappingUpsertInput, len(items))
	for i := range items {
		out[i] = MappingUpsertInput{
			BackendCategoryID: items[i].BackendCategoryID,
			RemoteCategoryID:  items[i].RemoteCategoryID,
			Enabled:           items[i].Enabled,
			SyncMode:          items[i].SyncMode,
			ConfirmedByStream: items[i].ConfirmedByStream,
		}
	}
	return out
}

func ToRemoteCategoryResponse(c RemoteCategory) RemoteCategoryResponse {
	return RemoteCategoryResponse{
		ID:                 c.ID,
		SiteID:             c.SiteID,
		RemoteTermID:       c.RemoteTermID,
		RemoteSlug:         c.RemoteSlug,
		RemoteName:         c.RemoteName,
		RemoteTaxonomy:     c.RemoteTaxonomy,
		ParentRemoteTermID: c.ParentRemoteTermID,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

func ToRemoteCategoryResponseList(items []RemoteCategory) []RemoteCategoryResponse {
	out := make([]RemoteCategoryResponse, len(items))
	for i := range items {
		out[i] = ToRemoteCategoryResponse(items[i])
	}
	return out
}

func ToMappingResponse(row MappingRow) MappingResponse {
	resp := MappingResponse{
		BackendCategory: BackendCategoryResponse{
			ID:   row.BackendCategory.ID,
			Name: row.BackendCategory.Name,
			Slug: row.BackendCategory.Slug,
		},
		Enabled:  false,
		SyncMode: SyncModeBlock,
		Status:   row.Status,
	}
	if row.Mapping != nil {
		resp.MappingID = &row.Mapping.ID
		resp.RemoteCategoryID = row.Mapping.RemoteCategoryID
		resp.Enabled = row.Mapping.Enabled
		resp.SyncMode = row.Mapping.SyncMode
		resp.CreatedBy = row.Mapping.CreatedBy
		resp.ConfirmedByWP = row.Mapping.ConfirmedByWP
		resp.ConfirmedByStream = row.Mapping.ConfirmedByStream
		resp.WPConfirmed = row.Mapping.ConfirmedByWP
		resp.StreamConfirmed = row.Mapping.ConfirmedByStream
		resp.MappingConfirmed = row.Mapping.ConfirmedByWP && row.Mapping.ConfirmedByStream && row.Mapping.RemoteCategoryID != nil
		resp.CreatedAt = &row.Mapping.CreatedAt
		resp.UpdatedAt = &row.Mapping.UpdatedAt
	}
	if row.RemoteCategory != nil {
		rc := ToRemoteCategoryResponse(*row.RemoteCategory)
		resp.RemoteCategory = &rc
	}
	return resp
}

func ToMappingResponseList(items []MappingRow) []MappingResponse {
	out := make([]MappingResponse, len(items))
	for i := range items {
		out[i] = ToMappingResponse(items[i])
	}
	return out
}
