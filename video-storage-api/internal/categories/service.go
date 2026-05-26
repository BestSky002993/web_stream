package categories

import (
	"context"
	"strings"
)

// Service holds the business rules: validation, slug normalization,
// default values, status whitelisting.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateParams is the service-facing input — the handler builds this
// from CreateCategoryRequest.
type CreateParams struct {
	Name        string
	Slug        string
	Description *string
	Status      string
	SortOrder   *int
}

// UpdateParams uses pointers to model PATCH semantics: nil = unchanged.
type UpdateParams struct {
	Name        *string
	Slug        *string
	Description *string
	Status      *string
	SortOrder   *int
}

// ListParams are the search/filter/page arguments accepted by List.
type ListParams struct {
	Page   int
	Limit  int
	Search string
	Status string
	Sort   string
}

// ListResult bundles the page slice with pagination metadata.
type ListResult struct {
	Items      []Category
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

// Create validates input, normalizes the slug, then inserts.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Category, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	slug := NormalizeSlug(p.Slug)
	if slug == "" {
		return nil, ErrInvalidSlug
	}

	status := p.Status
	if status == "" {
		status = StatusActive
	}
	if !isValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	sortOrder := 0
	if p.SortOrder != nil {
		sortOrder = *p.SortOrder
	}

	description := trimDescription(p.Description)

	return s.repo.Insert(ctx, CreateInput{
		Name:        name,
		Slug:        slug,
		Description: description,
		Status:      status,
		SortOrder:   sortOrder,
	})
}

// Get returns one category or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (*Category, error) {
	return s.repo.FindByID(ctx, id)
}

// Update applies a partial update. Each field is validated/normalized
// independently before reaching the repository.
func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Category, error) {
	// Ensure the row exists and is live before mutating — gives us a clean
	// ErrNotFound for already-deleted ids instead of a confusing "no rows
	// updated" path.
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, err
	}

	in := UpdateInput{}

	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return nil, ErrInvalidName
		}
		in.Name = &name
	}

	if p.Slug != nil {
		slug := NormalizeSlug(*p.Slug)
		if slug == "" {
			return nil, ErrInvalidSlug
		}
		in.Slug = &slug
	}

	if p.Status != nil {
		if !isValidStatus(*p.Status) {
			return nil, ErrInvalidStatus
		}
		in.Status = p.Status
	}

	if p.SortOrder != nil {
		in.SortOrder = p.SortOrder
	}

	if p.Description != nil {
		in.Description = trimDescription(p.Description)
	}

	return s.repo.Update(ctx, id, in)
}

// Delete soft-deletes the category.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}

// List returns a page plus pagination metadata. Defaults: page=1, limit=20.
// Limit is clamped to [1, 100].
func (s *Service) List(ctx context.Context, p ListParams) (*ListResult, error) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	limit := p.Limit
	switch {
	case limit < 1:
		limit = 20
	case limit > 100:
		limit = 100
	}

	status := p.Status
	if status != "" && !isValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	offset := (page - 1) * limit

	items, total, err := s.repo.List(ctx, ListFilters{
		Search: strings.TrimSpace(p.Search),
		Status: status,
		Sort:   p.Sort,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &ListResult{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func isValidStatus(s string) bool {
	return s == StatusActive || s == StatusDisabled
}

// trimDescription returns nil for nil or a pointer to the trimmed value
// otherwise. Empty after trim is preserved (so callers can "clear" the
// field by sending "").
func trimDescription(d *string) *string {
	if d == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*d)
	return &trimmed
}
