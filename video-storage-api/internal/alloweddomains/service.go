package alloweddomains

import (
	"context"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateParams struct {
	Domain string
	Type   string
	Status string
	Notes  *string
}

type UpdateParams struct {
	Domain *string
	Type   *string
	Status *string
	Notes  *string
}

type ListParams struct {
	Page   int
	Limit  int
	Search string
	Type   string
	Status string
}

type ListResult struct {
	Items      []Domain
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Domain, error) {
	domain, err := NormalizeDomain(p.Domain)
	if err != nil {
		return nil, err
	}
	t := p.Type
	if !IsValidType(t) {
		return nil, ErrInvalidType
	}
	status := p.Status
	if status == "" {
		status = StatusActive
	}
	if !IsValidStatus(status) {
		return nil, ErrInvalidStatus
	}
	notes := trimNotes(p.Notes)
	return s.repo.Insert(ctx, CreateInput{Domain: domain, Type: t, Status: status, Notes: notes})
}

func (s *Service) Get(ctx context.Context, id string) (*Domain, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Domain, error) {
	in := UpdateInput{Notes: trimNotes(p.Notes)}
	if p.Domain != nil {
		d, err := NormalizeDomain(*p.Domain)
		if err != nil {
			return nil, err
		}
		in.Domain = &d
	}
	if p.Type != nil {
		if !IsValidType(*p.Type) {
			return nil, ErrInvalidType
		}
		in.Type = p.Type
	}
	if p.Status != nil {
		if !IsValidStatus(*p.Status) {
			return nil, ErrInvalidStatus
		}
		in.Status = p.Status
	}
	return s.repo.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}

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
	if p.Type != "" && !IsValidType(p.Type) {
		return nil, ErrInvalidType
	}
	if p.Status != "" && !IsValidStatus(p.Status) {
		return nil, ErrInvalidStatus
	}
	items, total, err := s.repo.List(ctx, ListFilters{
		Search: p.Search,
		Type:   p.Type,
		Status: p.Status,
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		return nil, err
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return &ListResult{Items: items, Page: page, Limit: limit, Total: total, TotalPages: totalPages}, nil
}

func trimNotes(notes *string) *string {
	if notes == nil {
		return nil
	}
	n := strings.TrimSpace(*notes)
	if n == "" {
		return nil
	}
	return &n
}
