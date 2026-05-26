package sites

import "time"

// CreateSiteRequest is POST /api/v1/admin/sites.
type CreateSiteRequest struct {
	Name              string `json:"name"                binding:"required,min=1,max=200"`
	Domain            string `json:"domain"              binding:"required,min=1,max=253"`
	Status            string `json:"status"              binding:"omitempty,oneof=active disabled suspended"`
	DailyRequestLimit *int   `json:"daily_request_limit" binding:"omitempty"`
}

// UpdateSiteRequest is PUT /api/v1/admin/sites/:id.
type UpdateSiteRequest struct {
	Name              *string `json:"name"                binding:"omitempty,min=1,max=200"`
	Domain            *string `json:"domain"              binding:"omitempty,min=1,max=253"`
	Status            *string `json:"status"              binding:"omitempty,oneof=active disabled suspended"`
	DailyRequestLimit *int    `json:"daily_request_limit" binding:"omitempty"`
}

// SiteResponse is the canonical JSON shape for a site.
//
// `api_key` is populated ONLY on create / rotate-key responses — list,
// get, and update responses leave it empty. Plain keys never leave the
// API after the one-shot create/rotate response is delivered.
type SiteResponse struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Domain            string    `json:"domain"`
	Status            string    `json:"status"`
	DailyRequestLimit int       `json:"daily_request_limit"`
	APIKey            string    `json:"api_key,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ListSitesResponse is GET /api/v1/admin/sites.
type ListSitesResponse struct {
	Data       []SiteResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
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

func toResponse(s *Site) SiteResponse {
	return SiteResponse{
		ID:                s.ID,
		Name:              s.Name,
		Domain:            s.Domain,
		Status:            s.Status,
		DailyRequestLimit: s.DailyRequestLimit,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

func toResponseWithKey(s *Site, plainKey string) SiteResponse {
	r := toResponse(s)
	r.APIKey = plainKey
	return r
}

func toResponseList(ss []Site) []SiteResponse {
	out := make([]SiteResponse, len(ss))
	for i := range ss {
		out[i] = toResponse(&ss[i])
	}
	return out
}
