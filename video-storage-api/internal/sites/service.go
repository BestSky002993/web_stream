package sites

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/video-storage-api/internal/cache"
)

// Service implements the business rules for site management and
// authentication. It is the only layer that ever sees plain API keys
// (briefly, on Create / RotateAPIKey) — the repository only sees hashes.
type Service struct {
	repo  *Repository
	cache *cache.Client
	ttl   time.Duration
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetCache(c *cache.Client, ttl time.Duration) {
	s.cache = c
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	s.ttl = ttl
}

// CreateParams is the service input for Create.
type CreateParams struct {
	Name              string
	Domain            string
	Status            string
	DailyRequestLimit *int // optional; nil → default
}

// CreateResult bundles the persisted site with the freshly generated
// plain API key, which is returned to the admin exactly once.
type CreateResult struct {
	Site   *Site
	APIKey string
}

// Create validates input, generates a secure API key, hashes it, and
// inserts the row.
func (s *Service) Create(ctx context.Context, p CreateParams) (*CreateResult, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	domain, ok := normalizeHost(p.Domain)
	if !ok {
		return nil, ErrInvalidDomain
	}

	status := p.Status
	if status == "" {
		status = StatusActive
	}
	if !isValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	limit := DefaultDailyRequestLimit
	if p.DailyRequestLimit != nil {
		if *p.DailyRequestLimit < 0 {
			return nil, ErrInvalidLimit
		}
		limit = *p.DailyRequestLimit
	}

	plain, hash, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	site, err := s.repo.Insert(ctx, CreateInput{
		Name:              name,
		Domain:            domain,
		APIKeyHash:        hash,
		Status:            status,
		DailyRequestLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	return &CreateResult{Site: site, APIKey: plain}, nil
}

// Get returns one site by ID or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (*Site, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateParams is the service-facing partial update.
type UpdateParams struct {
	Name              *string
	Domain            *string
	Status            *string
	DailyRequestLimit *int
}

// Update applies a partial update.
func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Site, error) {
	in := UpdateInput{}

	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" {
			return nil, ErrInvalidName
		}
		in.Name = &n
	}

	if p.Domain != nil {
		d, ok := normalizeHost(*p.Domain)
		if !ok {
			return nil, ErrInvalidDomain
		}
		in.Domain = &d
	}

	if p.Status != nil {
		if !isValidStatus(*p.Status) {
			return nil, ErrInvalidStatus
		}
		in.Status = p.Status
	}

	if p.DailyRequestLimit != nil {
		if *p.DailyRequestLimit < 0 {
			return nil, ErrInvalidLimit
		}
		in.DailyRequestLimit = p.DailyRequestLimit
	}

	site, err := s.repo.Update(ctx, id, in)
	if err == nil {
		s.cacheSite(ctx, site)
	}
	return site, err
}

// Disable is a convenience for Update(status=disabled).
func (s *Service) Disable(ctx context.Context, id string) (*Site, error) {
	disabled := StatusDisabled
	site, err := s.repo.Update(ctx, id, UpdateInput{Status: &disabled})
	if err == nil {
		s.cacheSite(ctx, site)
	}
	return site, err
}

// Delete soft-deletes the site. The X-API-Key middleware will start
// rejecting requests using its key immediately (the row is gone from
// the live partial index).
func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.SoftDelete(ctx, id)
	if err == nil && s.cache != nil {
		s.cache.Delete(ctx, "site:id:"+id)
	}
	return err
}

// RotateAPIKey generates a new API key, replaces the stored hash, and
// returns the new plain key. The old key stops working immediately.
func (s *Service) RotateAPIKey(ctx context.Context, id string) (*CreateResult, error) {
	plain, hash, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}
	site, err := s.repo.UpdateAPIKey(ctx, id, hash)
	if err != nil {
		return nil, err
	}
	s.cacheSite(ctx, site)
	return &CreateResult{Site: site, APIKey: plain}, nil
}

// ListParams models the admin query string for List.
type ListParams struct {
	Page   int
	Limit  int
	Search string
	Status string
}

// ListResult bundles a page of sites with pagination metadata.
type ListResult struct {
	Items      []Site
	Page       int
	Limit      int
	Total      int
	TotalPages int
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

	if p.Status != "" && !isValidStatus(p.Status) {
		return nil, ErrInvalidStatus
	}

	items, total, err := s.repo.List(ctx, ListFilters{
		Search: p.Search,
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
	return &ListResult{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// Authenticate looks up a site by its plain API key. The key is hashed
// before the DB lookup; plain text is never compared to anything stored.
//
// Outcomes:
//   - empty / malformed input → ErrMissingAPIKey
//   - no match                → ErrInvalidAPIKey
//   - status='disabled'       → ErrSiteDisabled
//   - status='suspended'      → ErrSiteSuspended
//   - status='active'         → site
func (s *Service) Authenticate(ctx context.Context, plainKey string) (*Site, error) {
	plainKey = strings.TrimSpace(plainKey)
	if plainKey == "" {
		return nil, ErrMissingAPIKey
	}
	if !LooksLikeAPIKey(plainKey) {
		return nil, ErrInvalidAPIKey
	}

	hash := HashAPIKey(plainKey)
	if s.cache != nil {
		var cached Site
		if s.cache.GetJSON(ctx, "site:key:"+hash, &cached) {
			return s.authorizeCachedSite(&cached)
		}
	}
	site, err := s.repo.FindByAPIKeyHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, fmt.Errorf("lookup site by api key: %w", err)
	}

	s.cacheSite(ctx, site)
	return s.authorizeCachedSite(site)
}

func (s *Service) authorizeCachedSite(site *Site) (*Site, error) {
	switch site.Status {
	case StatusActive:
		return site, nil
	case StatusDisabled:
		return nil, ErrSiteDisabled
	case StatusSuspended:
		return nil, ErrSiteSuspended
	default:
		// Defensive — unknown statuses deny.
		return nil, ErrSiteDisabled
	}
}

func (s *Service) cacheSite(ctx context.Context, site *Site) {
	if s.cache == nil || site == nil {
		return
	}
	ttl := s.ttl
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	s.cache.SetJSON(ctx, "site:id:"+site.ID, site, ttl)
	s.cache.SetJSON(ctx, "site:key:"+site.APIKeyHash, site, minDuration(ttl, 60*time.Second))
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func isValidStatus(s string) bool {
	switch s {
	case StatusActive, StatusDisabled, StatusSuspended:
		return true
	}
	return false
}
