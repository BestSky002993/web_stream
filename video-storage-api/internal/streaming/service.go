package streaming

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/video-storage-api/internal/cache"
	"github.com/video-storage-api/internal/categorymappings"
	"github.com/video-storage-api/internal/publications"
	"github.com/video-storage-api/internal/storage"
	"github.com/video-storage-api/internal/videos"
)

// Service implements the streaming authorisation + lookup rules.
type Service struct {
	repo             *videos.Repository
	store            *storage.LocalStorage
	signer           *Signer
	policy           *DomainPolicy
	publications     *publications.Repository
	categoryMappings *categorymappings.Repository
	cache            *cache.Client
	cacheTTL         time.Duration
}

func NewService(repo *videos.Repository, store *storage.LocalStorage, signer *Signer, policy *DomainPolicy) *Service {
	return &Service{repo: repo, store: store, signer: signer, policy: policy}
}

// Signer exposes the underlying signer for the admin URL endpoint.
func (s *Service) Signer() *Signer { return s.signer }

// DomainPolicy exposes the domain whitelist.
func (s *Service) DomainPolicy() *DomainPolicy { return s.policy }

func (s *Service) SetPublications(repo *publications.Repository) {
	s.publications = repo
}

func (s *Service) SetCategoryMappings(repo *categorymappings.Repository) {
	s.categoryMappings = repo
}

func (s *Service) SetCache(c *cache.Client, ttl time.Duration) {
	s.cache = c
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	s.cacheTTL = ttl
}

// AccessParams carries the inputs needed to authorise a stream request.
type AccessParams struct {
	VideoID   string
	Origin    string
	Referer   string
	Signature string
	Expires   string // unix seconds as a string; empty means no signature
	SiteID    string
	Domain    string
	Nonce     string
}

// AuthorizeStream applies, in order:
//   - domain whitelist (Origin / Referer)
//   - lookup the video; reject if missing / deleted
//   - reject unless status='ready'
//   - require a valid signed URL for the media bytes
//
// Returns the resolved video so the handler can stream it.
func (s *Service) AuthorizeStream(ctx context.Context, p AccessParams) (*videos.Video, error) {
	if err := s.policy.Check(ctx, p.Origin, p.Referer); err != nil {
		return nil, err
	}

	v, err := s.findVideoByID(ctx, p.VideoID, 60*time.Second)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lookup video: %w", err)
	}

	if v.Status != videos.StatusReady {
		return nil, ErrNotAvailable
	}

	if p.SiteID != "" || p.Domain != "" || p.Nonce != "" {
		expires, err := parseExpires(p.Expires)
		if err != nil {
			return nil, err
		}
		if err := s.signer.VerifyClaims(SignatureClaims{
			Scope:   SignatureScope,
			VideoID: p.VideoID,
			SiteID:  p.SiteID,
			Domain:  p.Domain,
			Nonce:   p.Nonce,
			Expires: expires,
		}, p.Signature); err != nil {
			return nil, err
		}
		if err := checkSignedRequestDomain(p.Domain, p.Origin, p.Referer); err != nil {
			return nil, err
		}
	} else if err := s.signer.VerifyParams(p.VideoID, p.Signature, p.Expires); err != nil {
		return nil, err
	}
	return v, nil
}

// OpenForStream resolves the video's on-disk file and returns it along
// with its size. Caller is responsible for closing the file handle.
func (s *Service) OpenForStream(_ context.Context, v *videos.Video) (*os.File, int64, error) {
	f, err := s.store.OpenVideo(v.VideoFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, ErrFileMissing
		}
		return nil, 0, fmt.Errorf("open video file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, fmt.Errorf("stat video file: %w", err)
	}
	return f, info.Size(), nil
}

// PublicMetadataParams are the inputs to LookupPublicMetadata.
type PublicMetadataParams struct {
	Slug    string
	Origin  string
	Referer string
}

// LookupPublicMetadata returns metadata for a public, ready video by
// slug. Soft-deleted, draft/processing/blocked, and private videos all
// return ErrNotFound so the endpoint can't be used to enumerate them.
func (s *Service) LookupPublicMetadata(ctx context.Context, p PublicMetadataParams) (*videos.Video, error) {
	if err := s.policy.Check(ctx, p.Origin, p.Referer); err != nil {
		return nil, err
	}

	if s.cache != nil {
		var cached videos.Video
		if s.cache.GetJSON(ctx, "video:slug:"+p.Slug, &cached) {
			if cached.Visibility != videos.VisibilityPublic || cached.Status != videos.StatusReady {
				return nil, ErrNotFound
			}
			return &cached, nil
		}
	}

	v, err := s.repo.FindBySlug(ctx, p.Slug)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lookup video by slug: %w", err)
	}

	if v.Visibility != videos.VisibilityPublic || v.Status != videos.StatusReady {
		return nil, ErrNotFound
	}
	if s.cache != nil {
		s.cache.SetJSON(ctx, "video:slug:"+p.Slug, v, s.cacheTTL)
		s.cache.SetJSON(ctx, "video:id:"+v.ID, v, s.cacheTTL)
	}
	return v, nil
}

// PublicListParams models the query string for ListPublic.
type PublicListParams struct {
	Page       int
	Limit      int
	Search     string
	CategoryID string
	Sort       string
	Origin     string
	Referer    string
	SiteID     string
}

// PublicListResult bundles a page of public videos with pagination metadata.
type PublicListResult struct {
	Items      []videos.Video
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

// ListPublic returns public, ready videos. visibility=public + status=ready +
// non-deleted are enforced at the repository level; the public client
// has no way to widen them.
func (s *Service) ListPublic(ctx context.Context, p PublicListParams) (*PublicListResult, error) {
	if err := s.policy.Check(ctx, p.Origin, p.Referer); err != nil {
		return nil, err
	}

	page := p.Page
	if page < 1 {
		page = 1
	}
	limit := ClampLimit(p.Limit, 20, 100)
	offset := (page - 1) * limit
	cacheKey := fmt.Sprintf("videos:list:site=%s:page=%d:limit=%d:search=%s:cat=%s:sort=%s", p.SiteID, page, limit, p.Search, p.CategoryID, p.Sort)
	if s.cache != nil {
		var cached PublicListResult
		if s.cache.GetJSON(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}

	var (
		items []videos.Video
		total int
		err   error
	)
	if p.SiteID != "" {
		if s.publications == nil {
			return nil, fmt.Errorf("site publications repository is not configured")
		}
		pubs, pubTotal, err := s.publications.ListApprovedVideos(ctx, p.SiteID, videos.ListPublicFilters{
			Search:     p.Search,
			CategoryID: p.CategoryID,
			Sort:       p.Sort,
			Limit:      limit,
			Offset:     offset,
		})
		if err != nil {
			return nil, fmt.Errorf("list approved videos: %w", err)
		}
		items = videosFromPublications(pubs)
		total = pubTotal
	} else {
		items, total, err = s.repo.ListPublic(ctx, videos.ListPublicFilters{
			Search:     p.Search,
			CategoryID: p.CategoryID,
			Sort:       p.Sort,
			Limit:      limit,
			Offset:     offset,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("list public videos: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	result := &PublicListResult{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
	if s.cache != nil {
		s.cache.SetJSON(ctx, cacheKey, result, minDuration(s.cacheTTL, 2*time.Minute))
	}
	return result, nil
}

// RandomPublicParams models the query string for RandomPublic.
type RandomPublicParams struct {
	Limit   int
	Seed    string
	Origin  string
	Referer string
	SiteID  string
}

// RandomPublicResult bundles the random feed with the seed key actually
// used (which the caller can echo back in the response for transparency).
type RandomPublicResult struct {
	Items   []videos.Video
	SeedKey string
}

// RandomPublic returns up to `limit` random public ready videos. If
// Seed is non-empty, the order is stable for the same seed within the
// same UTC day; the daily-rotating per-site feed pattern from the spec.
func (s *Service) RandomPublic(ctx context.Context, p RandomPublicParams) (*RandomPublicResult, error) {
	if err := s.policy.Check(ctx, p.Origin, p.Referer); err != nil {
		return nil, err
	}

	limit := ClampLimit(p.Limit, 20, 50)
	seedKey := BuildSeedKey(p.Seed, time.Now())
	cacheKey := fmt.Sprintf("videos:random:site=%s:limit=%d:seed=%s", p.SiteID, limit, seedKey)
	if s.cache != nil {
		var cached RandomPublicResult
		if s.cache.GetJSON(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}

	var (
		items []videos.Video
		err   error
	)
	if p.SiteID != "" {
		if s.publications == nil {
			return nil, fmt.Errorf("site publications repository is not configured")
		}
		pubs, err := s.publications.RandomApprovedVideos(ctx, p.SiteID, limit, seedKey)
		if err != nil {
			return nil, fmt.Errorf("random approved videos: %w", err)
		}
		items = videosFromPublications(pubs)
	} else {
		items, err = s.repo.RandomPublic(ctx, limit, seedKey)
	}
	if err != nil {
		return nil, fmt.Errorf("random public videos: %w", err)
	}
	result := &RandomPublicResult{
		Items:   items,
		SeedKey: seedKey,
	}
	if s.cache != nil {
		s.cache.SetJSON(ctx, cacheKey, result, minDuration(s.cacheTTL, 2*time.Minute))
	}
	return result, nil
}

func (s *Service) ListSitePublications(ctx context.Context, siteID string, p PublicListParams) ([]publications.Publication, int, error) {
	if s.publications == nil {
		return nil, 0, fmt.Errorf("site publications repository is not configured")
	}
	page := p.Page
	if page < 1 {
		page = 1
	}
	limit := ClampLimit(p.Limit, 20, 100)
	return s.publications.ListApprovedVideos(ctx, siteID, videos.ListPublicFilters{
		Search:     p.Search,
		CategoryID: p.CategoryID,
		Sort:       p.Sort,
		Limit:      limit,
		Offset:     (page - 1) * limit,
	})
}

func (s *Service) MappingStatusForPublications(ctx context.Context, siteID string, pubs []publications.Publication) (map[string]categorymappings.MappingRow, error) {
	if s.categoryMappings == nil {
		return map[string]categorymappings.MappingRow{}, nil
	}
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(pubs))
	for i := range pubs {
		if pubs[i].Video == nil || pubs[i].Video.CategoryID == "" {
			continue
		}
		if _, ok := seen[pubs[i].Video.CategoryID]; ok {
			continue
		}
		seen[pubs[i].Video.CategoryID] = struct{}{}
		ids = append(ids, pubs[i].Video.CategoryID)
	}
	cacheKey := fmt.Sprintf("category_mappings:site=%s:ids=%s", siteID, strings.Join(ids, ","))
	if s.cache != nil {
		var cached map[string]categorymappings.MappingRow
		if s.cache.GetJSON(ctx, cacheKey, &cached) {
			return cached, nil
		}
	}
	rows, err := s.categoryMappings.MappingStatusForCategories(ctx, siteID, ids)
	if err == nil && s.cache != nil {
		s.cache.SetJSON(ctx, cacheKey, rows, minDuration(s.cacheTTL, 2*time.Minute))
	}
	return rows, err
}

func (s *Service) ListSyncCategories(ctx context.Context) ([]publications.PublicationCategory, error) {
	if s.publications == nil {
		return nil, fmt.Errorf("site publications repository is not configured")
	}
	return s.publications.ListCategories(ctx)
}

func (s *Service) UpdateSyncResult(ctx context.Context, siteID, videoID, status string, remotePostID, remoteSlug *string) (*publications.Publication, error) {
	if s.publications == nil {
		return nil, fmt.Errorf("site publications repository is not configured")
	}
	return s.publications.UpdateSyncResult(ctx, siteID, videoID, status, remotePostID, remoteSlug)
}

func videosFromPublications(pubs []publications.Publication) []videos.Video {
	out := make([]videos.Video, 0, len(pubs))
	for i := range pubs {
		if pubs[i].Video != nil {
			out = append(out, *pubs[i].Video)
		}
	}
	return out
}

func (s *Service) findVideoByID(ctx context.Context, id string, ttl time.Duration) (*videos.Video, error) {
	if s.cache != nil {
		var cached videos.Video
		if s.cache.GetJSON(ctx, "video:id:"+id, &cached) {
			return &cached, nil
		}
	}
	v, err := s.repo.FindByID(ctx, id)
	if err == nil && s.cache != nil {
		s.cache.SetJSON(ctx, "video:id:"+id, v, minDuration(s.cacheTTL, ttl))
	}
	return v, err
}

func minDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}

// ThumbnailParams holds inputs for OpenThumbnailForVideo.
type ThumbnailParams struct {
	VideoID string
	Origin  string
	Referer string
}

// OpenThumbnailForVideo serves the thumbnail. Thumbnails are always
// served regardless of visibility — they're low-value preview frames
// and matching this to industry behaviour avoids needing a signed URL
// for every <img>.
//
// We still enforce the domain whitelist so it can't be hotlinked into
// random sites that haven't been authorised.
func (s *Service) OpenThumbnailForVideo(ctx context.Context, p ThumbnailParams) (*videos.Video, *os.File, int64, error) {
	if err := s.policy.Check(ctx, p.Origin, p.Referer); err != nil {
		return nil, nil, 0, err
	}

	v, err := s.repo.FindByID(ctx, p.VideoID)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, nil, 0, ErrNotFound
		}
		return nil, nil, 0, fmt.Errorf("lookup video: %w", err)
	}

	f, err := s.store.OpenThumbnail(v.ThumbnailFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, 0, ErrFileMissing
		}
		return nil, nil, 0, fmt.Errorf("open thumbnail: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, 0, fmt.Errorf("stat thumbnail: %w", err)
	}
	return v, f, info.Size(), nil
}

// SignedURL contains the bits an admin needs to embed a stream link.
type SignedURL struct {
	URL       string
	Expires   int64
	ExpiresAt time.Time
}

// IssueSignedURL generates a signed stream URL for any ready video,
// regardless of visibility. Used by the admin endpoint.
func (s *Service) IssueSignedURL(ctx context.Context, videoID, baseURL string) (*SignedURL, error) {
	v, err := s.repo.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lookup video: %w", err)
	}
	if v.Status != videos.StatusReady {
		return nil, ErrNotAvailable
	}
	return s.buildSignedURL(v.ID, baseURL), nil
}

// IssuePublicSignedURL generates a signed stream URL only when the video
// is public and ready. Used by the site (X-API-Key) endpoint so a site
// cannot use a known video ID to bypass the admin-only visibility split.
//
// Non-public or non-ready videos return ErrNotFound (not ErrNotAvailable)
// so the site can't enumerate private content.
func (s *Service) IssuePublicSignedURL(ctx context.Context, siteID, siteDomain, videoID, baseURL string) (*SignedURL, error) {
	v, err := s.repo.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lookup video: %w", err)
	}
	if v.Status != videos.StatusReady || v.Visibility != videos.VisibilityPublic {
		return nil, ErrNotFound
	}
	if s.publications == nil {
		return nil, fmt.Errorf("site publications repository is not configured")
	}
	approved, err := s.publications.IsPlayableForSite(ctx, siteID, videoID)
	if err != nil {
		return nil, err
	}
	if !approved {
		return nil, ErrNotFound
	}
	return s.buildSiteSignedURL(v.ID, siteID, siteDomain, baseURL), nil
}

func (s *Service) buildSignedURL(videoID, baseURL string) *SignedURL {
	sig, expires := s.signer.IssueNow(videoID)
	url := fmt.Sprintf("%s/api/v1/stream/%s?expires=%d&signature=%s",
		baseURL, videoID, expires, sig)
	return &SignedURL{
		URL:       url,
		Expires:   expires,
		ExpiresAt: time.Unix(expires, 0).UTC(),
	}
}

func (s *Service) buildSiteSignedURL(videoID, siteID, domain, baseURL string) *SignedURL {
	sig, claims := s.signer.IssueClaimsNow(SignatureScope, videoID, siteID, domain)
	url := fmt.Sprintf("%s/api/v1/stream/%s?expires=%d&site_id=%s&domain=%s&nonce=%s&signature=%s",
		baseURL, videoID, claims.Expires, claims.SiteID, claims.Domain, claims.Nonce, sig)
	return &SignedURL{
		URL:       url,
		Expires:   claims.Expires,
		ExpiresAt: time.Unix(claims.Expires, 0).UTC(),
	}
}
