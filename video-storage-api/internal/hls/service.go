package hls

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/video-storage-api/internal/cache"
	"github.com/video-storage-api/internal/publications"
	"github.com/video-storage-api/internal/streaming"
	"github.com/video-storage-api/internal/videos"
)

// Service ties together the processor, repository, signer, and the
// videos repo (read-only) to satisfy the URL-issuance flow.
type Service struct {
	processor    *Processor
	repo         *Repository
	videosRepo   *videos.Repository
	signer       *streaming.Signer
	publications *publications.Repository
	cache        *cache.Client
	cacheTTL     time.Duration
}

func NewService(processor *Processor, repo *Repository, videosRepo *videos.Repository, signer *streaming.Signer) *Service {
	return &Service{
		processor:  processor,
		repo:       repo,
		videosRepo: videosRepo,
		signer:     signer,
	}
}

func (s *Service) SetPublications(repo *publications.Repository) {
	s.publications = repo
}

func (s *Service) SetCache(c *cache.Client, ttl time.Duration) {
	s.cache = c
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	s.cacheTTL = ttl
}

// Start kicks off an HLS conversion in a fresh goroutine and returns
// immediately. Implements videos.HLSStarter.
//
// Status transitions:
//   - mark hls_status='processing' before launching ffmpeg
//   - on success: hls_status='ready' + hls_ready_at=NOW()
//   - on failure: hls_status='failed' + hls_error=<truncated message>
func (s *Service) Start(videoID, sourceFilename string) {
	if err := ValidateVideoID(videoID); err != nil {
		log.Printf("hls: refusing to start with invalid videoID %q: %v", videoID, err)
		return
	}

	go func() {
		// Each job gets its own context so the request-scoped lifetime
		// doesn't yank ffmpeg mid-encode.
		ctx, cancel := context.WithTimeout(context.Background(), s.processor.cfg.ConvertTimeout+time.Minute)
		defer cancel()

		if err := s.repo.SetProcessing(ctx, videoID); err != nil {
			log.Printf("hls: failed to mark processing for %s: %v", videoID, err)
			return
		}

		variants, err := s.processor.ConvertToHLS(ctx, videoID, sourceFilename)
		if err != nil {
			log.Printf("hls: conversion failed for %s: %v", videoID, err)
			if dberr := s.repo.SetFailed(ctx, videoID, err.Error()); dberr != nil {
				log.Printf("hls: also failed to record failure for %s: %v", videoID, dberr)
			}
			return
		}

		if err := s.repo.SetReady(ctx, videoID, variants); err != nil {
			log.Printf("hls: conversion finished but DB update failed for %s: %v", videoID, err)
			// Don't roll back the on-disk output — the next admin refresh
			// will still show a stale 'processing' status but playback
			// won't work anyway without the row update. A subsequent
			// retry will succeed once the DB recovers.
			return
		}
		log.Printf("hls: conversion ready for %s (variants=%v)", videoID, variants)
	}()
}

// SignedURL is what the URL issuer hands back to the dashboard / WP.
type SignedURL struct {
	URL       string
	Expires   int64
	ExpiresAt time.Time
}

// IssueAdminURL signs a master.m3u8 URL for any video the admin can
// see. Returns ErrNotReady if HLS conversion hasn't completed yet.
func (s *Service) IssueAdminURL(ctx context.Context, videoID, baseURL string) (*SignedURL, error) {
	v, err := s.videosRepo.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("hls: lookup video: %w", err)
	}
	if v.HLSStatus != StatusReady {
		return nil, ErrNotReady
	}
	return s.buildSignedURL(videoID, baseURL), nil
}

// IssueSiteURL is the site-key counterpart — restricted to public + ready
// videos so a site can't probe pipeline state on private content.
func (s *Service) IssueSiteURL(ctx context.Context, siteID, siteDomain, videoID, baseURL string) (*SignedURL, error) {
	v, err := s.videosRepo.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, videos.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("hls: lookup video: %w", err)
	}
	// Hide non-public videos behind a 404 even when HLS is ready.
	if v.Visibility != videos.VisibilityPublic || v.Status != videos.StatusReady {
		return nil, ErrNotFound
	}
	if v.HLSStatus != StatusReady {
		return nil, ErrNotReady
	}
	if s.publications == nil {
		return nil, fmt.Errorf("hls: site publications repository is not configured")
	}
	approved, err := s.publications.IsPlayableForSite(ctx, siteID, videoID)
	if err != nil {
		return nil, err
	}
	if !approved {
		return nil, ErrNotFound
	}
	return s.buildSiteSignedURL(videoID, siteID, siteDomain, baseURL), nil
}

func (s *Service) buildSignedURL(videoID, baseURL string) *SignedURL {
	sig, expires := s.signer.IssueNowScoped(streaming.SignatureScopeHLS, videoID)
	url := fmt.Sprintf("%s/api/v1/hls/%s/%s?expires=%d&signature=%s",
		baseURL, videoID, MasterFilename, expires, sig)
	return &SignedURL{
		URL:       url,
		Expires:   expires,
		ExpiresAt: time.Unix(expires, 0).UTC(),
	}
}

func (s *Service) buildSiteSignedURL(videoID, siteID, domain, baseURL string) *SignedURL {
	sig, claims := s.signer.IssueClaimsNow(streaming.SignatureScopeHLS, videoID, siteID, domain)
	url := fmt.Sprintf("%s/api/v1/hls/%s/%s?expires=%d&site_id=%s&domain=%s&nonce=%s&signature=%s",
		baseURL, videoID, MasterFilename, claims.Expires, claims.SiteID, claims.Domain, claims.Nonce, sig)
	return &SignedURL{
		URL:       url,
		Expires:   claims.Expires,
		ExpiresAt: time.Unix(claims.Expires, 0).UTC(),
	}
}

// VerifyParams runs the HLS-scoped signature check used by every
// manifest / segment serving handler.
func (s *Service) VerifyParams(videoID, signature, expiresRaw string) error {
	return s.signer.VerifyParamsScoped(streaming.SignatureScopeHLS, videoID, signature, expiresRaw)
}

func (s *Service) VerifySiteParams(videoID, signature, expiresRaw, siteID, domain, nonce, origin, referer string) error {
	if siteID == "" && domain == "" && nonce == "" {
		return s.VerifyParams(videoID, signature, expiresRaw)
	}
	expires, err := streaming.ParseExpiresForHLS(expiresRaw)
	if err != nil {
		return err
	}
	if err := s.signer.VerifyClaims(streaming.SignatureClaims{
		Scope:   streaming.SignatureScopeHLS,
		VideoID: videoID,
		SiteID:  siteID,
		Domain:  domain,
		Nonce:   nonce,
		Expires: expires,
	}, signature); err != nil {
		return err
	}
	return streaming.CheckSignedRequestDomainForHLS(domain, origin, referer)
}

// Paths returns the path helpers so the HTTP layer can resolve files.
func (s *Service) Paths() *Paths { return s.processor.paths }

func (s *Service) CachedBytes(ctx context.Context, key string) ([]byte, bool) {
	if s.cache == nil {
		return nil, false
	}
	b, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, false
	}
	return b, true
}

func (s *Service) CacheBytes(ctx context.Context, key string, value []byte, ttl time.Duration) {
	if s.cache == nil {
		return
	}
	if ttl <= 0 {
		ttl = s.cacheTTL
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if err := s.cache.Set(ctx, key, value, ttl); err != nil {
		log.Printf("[CACHE] event=set_failed key=%s err=%v", key, err)
	}
}
