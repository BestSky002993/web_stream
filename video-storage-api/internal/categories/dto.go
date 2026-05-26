package categories

import "time"

// CreateCategoryRequest is the JSON body for POST /api/v1/admin/categories.
type CreateCategoryRequest struct {
	Name        string  `json:"name"        binding:"required,min=1,max=200"`
	Slug        string  `json:"slug"        binding:"required,min=1,max=200"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Status      string  `json:"status"      binding:"omitempty,oneof=active disabled"`
	SortOrder   *int    `json:"sort_order"  binding:"omitempty"`
}

// UpdateCategoryRequest is the JSON body for PUT /api/v1/admin/categories/:id.
// All fields are optional: a nil pointer means "leave unchanged".
type UpdateCategoryRequest struct {
	Name        *string `json:"name"        binding:"omitempty,min=1,max=200"`
	Slug        *string `json:"slug"        binding:"omitempty,min=1,max=200"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Status      *string `json:"status"      binding:"omitempty,oneof=active disabled"`
	SortOrder   *int    `json:"sort_order"  binding:"omitempty"`
}

// CategoryResponse is the canonical JSON projection of a Category.
type CategoryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListCategoriesResponse is what GET /api/v1/admin/categories returns.
type ListCategoriesResponse struct {
	Data       []CategoryResponse `json:"data"`
	Pagination Pagination         `json:"pagination"`
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

func toResponse(c *Category) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		Status:      c.Status,
		SortOrder:   c.SortOrder,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toResponseList(cs []Category) []CategoryResponse {
	out := make([]CategoryResponse, len(cs))
	for i := range cs {
		out[i] = toResponse(&cs[i])
	}
	return out
}
