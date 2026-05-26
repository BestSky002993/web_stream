package videos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/video-storage-api/internal/storage"
)

// HLSStarter kicks off an HLS conversion job for a freshly inserted
// video. It's a tiny interface so the videos service has no compile-
// time dependency on the hls package — `Start` returns immediately
// and the actual conversion runs in the background.
type HLSStarter interface {
	Start(videoID, sourceFilename string)
}

// Service implements the upload + CRUD business rules. It is the only
// place where filesystem, ffmpeg, and database operations are sequenced
// together — so it also owns rollback (file cleanup on DB failure) and
// orphan replacement (deleting the old thumbnail after a successful
// regeneration).
type Service struct {
	repo  *Repository
	store *storage.LocalStorage
	maxV  int64      // max video size in bytes
	hls   HLSStarter // optional — nil disables HLS conversion
}

func NewService(repo *Repository, store *storage.LocalStorage, maxVideoBytes int64) *Service {
	return &Service{
		repo:  repo,
		store: store,
		maxV:  maxVideoBytes,
	}
}

// SetHLSStarter attaches the HLS pipeline kickoff. Wired from routes.go
// after both modules are constructed — keeps videos→hls dependency-free
// at the package level.
func (s *Service) SetHLSStarter(starter HLSStarter) {
	s.hls = starter
}

// UploadFile is the minimal handle the service needs from a multipart
// part. The handler builds these from *multipart.FileHeader; tests can
// build them directly.
type UploadFile struct {
	OriginalName string
	Size         int64
	Open         func() (io.ReadCloser, error)
}

// CreateParams is the service-facing input for upload.
//
// ThumbnailSecond is required at this layer (defaulted by the handler
// to DefaultThumbnailSecond when the client omits it).
type CreateParams struct {
	Title             string
	Slug              string
	Description       string
	CategoryID        string
	Visibility        string
	ThumbnailSecond   int
	UploadedByAdminID string
	Video             UploadFile
}

// Create runs the full upload pipeline:
//
//  1. validate text fields and look up the category
//  2. sniff + size-check the video file
//  3. stream the video into private storage
//  4. extract a thumbnail frame via ffmpeg at ThumbnailSecond
//  5. insert the DB row with status='ready'
//
// On any failure after step 3, files written by earlier steps are
// removed so nothing is orphaned.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Video, error) {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		return nil, ErrInvalidTitle
	}

	slug := NormalizeSlug(p.Slug)
	if slug == "" {
		return nil, ErrInvalidSlug
	}

	visibility := p.Visibility
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		return nil, ErrInvalidVisibility
	}

	if p.ThumbnailSecond < 0 || p.ThumbnailSecond > storage.MaxThumbnailSecond {
		return nil, ErrInvalidThumbnailSecond
	}

	exists, err := s.repo.CategoryExists(ctx, p.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("category check: %w", err)
	}
	if !exists {
		return nil, ErrCategoryNotFound
	}

	if p.Video.Size > s.maxV {
		return nil, ErrVideoTooLarge
	}

	videoFilename, videoSize, videoMime, err := s.storeVideoUpload(p.Video)
	if err != nil {
		return nil, err
	}

	// Probe duration once at upload time so subsequent edits can clamp
	// without re-probing. ffprobe failure is non-fatal — duration is
	// nullable and the thumbnail extractor still works on best-effort.
	duration, durationOK := s.store.ProbeDuration(ctx, videoFilename)

	thumbFilename, actualSecond, err := s.store.GenerateThumbnailFromVideo(ctx, videoFilename, p.ThumbnailSecond, duration)
	if err != nil {
		if delErr := s.store.DeleteVideo(videoFilename); delErr != nil {
			log.Printf("videos: cleanup video after thumbnail-gen failure: %v", delErr)
		}
		return nil, mapStorageError(err)
	}

	desc := trimDescription(p.Description)

	var durationPtr *int
	if durationOK && duration > 0 {
		durationPtr = &duration
	}

	v, err := s.repo.Insert(ctx, InsertInput{
		CategoryID:        p.CategoryID,
		Title:             title,
		Slug:              slug,
		Description:       desc,
		Status:            StatusReady,
		Visibility:        visibility,
		VideoFilename:     videoFilename,
		VideoOriginalName: sanitizeOriginalName(p.Video.OriginalName),
		VideoMimeType:     videoMime,
		VideoSizeBytes:    videoSize,
		ThumbnailFilename: thumbFilename,
		ThumbnailSecond:   actualSecond, // store the clamped value, not the request
		DurationSeconds:   durationPtr,
		UploadedByAdminID: p.UploadedByAdminID,
	})
	if err != nil {
		if delErr := s.store.DeleteVideo(videoFilename); delErr != nil {
			log.Printf("videos: cleanup video after db failure: %v", delErr)
		}
		if delErr := s.store.DeleteThumbnail(thumbFilename); delErr != nil {
			log.Printf("videos: cleanup thumbnail after db failure: %v", delErr)
		}
		return nil, err
	}

	// Kick off HLS conversion asynchronously — the upload response
	// returns immediately, and the caller polls `hls.status` (or the
	// dashboard refreshes the row) to see progress.
	if s.hls != nil {
		s.hls.Start(v.ID, v.VideoFilename)
	}

	return v, nil
}

// storeVideoUpload sniffs the video's MIME against the allow list and
// streams it into the videos directory.
func (s *Service) storeVideoUpload(u UploadFile) (string, int64, string, error) {
	if u.Open == nil {
		return "", 0, "", ErrMissingVideo
	}

	rc, err := u.Open()
	if err != nil {
		return "", 0, "", fmt.Errorf("open upload: %w", err)
	}
	defer rc.Close()

	mime, replay, err := storage.SniffMIME(rc)
	if err != nil {
		return "", 0, "", fmt.Errorf("sniff mime: %w", err)
	}

	ext, ok := storage.ResolveExt(storage.KindVideo, mime)
	if !ok {
		return "", 0, "", ErrUnsupportedVideoType
	}

	filename, size, err := s.store.SaveVideo(replay, ext)
	if err != nil {
		return "", 0, "", fmt.Errorf("save upload: %w", err)
	}

	return filename, size, mime, nil
}

// Get returns one video, joining category, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (*Video, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateParams is the service-facing partial-update shape.
// Empty strings in non-pointer fields mean "no change". A non-nil
// ThumbnailSecond triggers a thumbnail regeneration.
type UpdateParams struct {
	Title           string
	Slug            string
	Description     string
	CategoryID      string
	Status          string
	Visibility      string
	ThumbnailSecond *int
}

// Update applies a partial update. If ThumbnailSecond is non-nil, the
// service runs ffmpeg against the existing video file, stores the new
// thumbnail, then runs the DB UPDATE; the old thumbnail file is only
// deleted after a successful commit.
func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Video, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	in := UpdateInput{}

	if v := strings.TrimSpace(p.Title); v != "" {
		in.Title = &v
	}

	if p.Slug != "" {
		slug := NormalizeSlug(p.Slug)
		if slug == "" {
			return nil, ErrInvalidSlug
		}
		in.Slug = &slug
	}

	if p.Description != "" {
		desc := strings.TrimSpace(p.Description)
		in.Description = &desc
	}

	if p.CategoryID != "" {
		exists, err := s.repo.CategoryExists(ctx, p.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("category check: %w", err)
		}
		if !exists {
			return nil, ErrCategoryNotFound
		}
		in.CategoryID = &p.CategoryID
	}

	if p.Status != "" {
		if !isValidUserStatus(p.Status) {
			return nil, ErrInvalidStatus
		}
		in.Status = &p.Status
	}

	if p.Visibility != "" {
		if p.Visibility != VisibilityPublic && p.Visibility != VisibilityPrivate {
			return nil, ErrInvalidVisibility
		}
		in.Visibility = &p.Visibility
	}

	// Thumbnail regeneration. We always regenerate when thumbnail_second
	// is provided in the request, even if it equals the current value —
	// that gives admins a way to retry if a previous extraction failed.
	//
	// The storage layer clamps the requested second against the known
	// duration. If the existing record doesn't have a duration yet
	// (uploaded before Phase 12), probe lazily here and persist it.
	var newThumbFilename string
	if p.ThumbnailSecond != nil {
		sec := *p.ThumbnailSecond
		if sec < 0 || sec > storage.MaxThumbnailSecond {
			return nil, ErrInvalidThumbnailSecond
		}

		duration := 0
		var lazyProbed *int
		if existing.DurationSeconds != nil {
			duration = *existing.DurationSeconds
		} else if d, ok := s.store.ProbeDuration(ctx, existing.VideoFilename); ok {
			duration = d
			lazyProbed = &d
		}

		fn, actualSecond, err := s.store.GenerateThumbnailFromVideo(ctx, existing.VideoFilename, sec, duration)
		if err != nil {
			return nil, mapStorageError(err)
		}
		newThumbFilename = fn
		in.ThumbnailFilename = &newThumbFilename
		in.ThumbnailSecond = &actualSecond
		if lazyProbed != nil {
			in.DurationSeconds = lazyProbed
		}
	}

	updated, err := s.repo.Update(ctx, id, in)
	if err != nil {
		if newThumbFilename != "" {
			if delErr := s.store.DeleteThumbnail(newThumbFilename); delErr != nil {
				log.Printf("videos: cleanup new thumbnail after update failure: %v", delErr)
			}
		}
		return nil, err
	}

	// Update committed — remove the displaced thumbnail.
	if newThumbFilename != "" && existing.ThumbnailFilename != "" && existing.ThumbnailFilename != newThumbFilename {
		if err := s.store.DeleteThumbnail(existing.ThumbnailFilename); err != nil {
			log.Printf("videos: failed to remove old thumbnail %q: %v", existing.ThumbnailFilename, err)
		}
	}

	return updated, nil
}

// Delete soft-deletes the video without touching stored files.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}

// ListParams models the query string for List.
type ListParams struct {
	Page              int
	Limit             int
	Search            string
	CategoryID        string
	Status            string
	Visibility        string
	NotApprovedSiteID string
	Sort              string
}

// ListResult bundles a page slice with pagination metadata.
type ListResult struct {
	Items      []Video
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

	if p.Status != "" && !isValidStatusForListFilter(p.Status) {
		return nil, ErrInvalidStatus
	}
	if p.Visibility != "" && p.Visibility != VisibilityPublic && p.Visibility != VisibilityPrivate {
		return nil, ErrInvalidVisibility
	}
	if p.NotApprovedSiteID != "" && !uuidPattern.MatchString(p.NotApprovedSiteID) {
		return nil, ErrInvalidSiteID
	}

	offset := (page - 1) * limit

	items, total, err := s.repo.List(ctx, ListFilters{
		Search:            p.Search,
		CategoryID:        p.CategoryID,
		Status:            p.Status,
		Visibility:        p.Visibility,
		NotApprovedSiteID: p.NotApprovedSiteID,
		Sort:              p.Sort,
		Limit:             limit,
		Offset:            offset,
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

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidUserStatus is the whitelist for status values an admin may set
// via Update. 'deleted' is excluded — that transition only happens via
// the DELETE endpoint.
func isValidUserStatus(s string) bool {
	switch s {
	case StatusDraft, StatusProcessing, StatusReady, StatusBlocked:
		return true
	}
	return false
}

// isValidStatusForListFilter accepts every defined status.
func isValidStatusForListFilter(s string) bool {
	switch s {
	case StatusDraft, StatusProcessing, StatusReady, StatusBlocked, StatusDeleted:
		return true
	}
	return false
}

// trimDescription returns nil for empty input so NULL is stored in the DB.
func trimDescription(d string) *string {
	d = strings.TrimSpace(d)
	if d == "" {
		return nil
	}
	return &d
}

// sanitizeOriginalName strips control characters and clamps the length
// of a client-supplied filename so it's safe to persist as display text.
func sanitizeOriginalName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "upload"
	}

	var b strings.Builder
	for _, r := range name {
		if r >= 0x20 && r != 0x7f {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		out = "upload"
	}
	if len(out) > 255 {
		out = out[:255]
	}
	return out
}

// mapStorageError translates storage-layer ffmpeg errors into the
// service's typed errors so the handler can pick the right HTTP code.
func mapStorageError(err error) error {
	switch {
	case errors.Is(err, storage.ErrInvalidThumbnailSecond):
		return ErrInvalidThumbnailSecond
	case errors.Is(err, storage.ErrFFmpegFailed):
		log.Printf("videos: ffmpeg failure: %v", err)
		return ErrThumbnailGeneration
	default:
		return err
	}
}
