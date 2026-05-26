package alloweddomains

import "time"

type CreateRequest struct {
	Domain string  `json:"domain" binding:"required,min=1,max=253"`
	Type   string  `json:"type" binding:"required,oneof=stream cors both"`
	Status string  `json:"status" binding:"omitempty,oneof=active disabled"`
	Notes  *string `json:"notes" binding:"omitempty,max=2000"`
}

type UpdateRequest struct {
	Domain *string `json:"domain" binding:"omitempty,min=1,max=253"`
	Type   *string `json:"type" binding:"omitempty,oneof=stream cors both"`
	Status *string `json:"status" binding:"omitempty,oneof=active disabled"`
	Notes  *string `json:"notes" binding:"omitempty,max=2000"`
}

type Response struct {
	ID        string     `json:"id"`
	Domain    string     `json:"domain"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Notes     *string    `json:"notes"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type ListResponse struct {
	Data       []Response `json:"data"`
	Pagination Pagination `json:"pagination"`
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

func toResponse(d *Domain) Response {
	return Response{
		ID:        d.ID,
		Domain:    d.Domain,
		Type:      d.Type,
		Status:    d.Status,
		Notes:     d.Notes,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		DeletedAt: d.DeletedAt,
	}
}

func toResponseList(ds []Domain) []Response {
	out := make([]Response, len(ds))
	for i := range ds {
		out[i] = toResponse(&ds[i])
	}
	return out
}
