package publications

import (
	"context"
	"regexp"

	"github.com/video-storage-api/internal/categorymappings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type ListParams struct {
	SiteID  string
	VideoID string
	Status  string
	Page    int
	Limit   int
}

func (s *Service) Upsert(ctx context.Context, in UpsertInput) (*Publication, error) {
	if !IsValidStatus(in.Status) {
		return nil, ErrInvalidStatus
	}
	if in.SyncMode != nil && !categorymappings.IsValidSyncMode(*in.SyncMode) {
		return nil, ErrInvalidSyncMode
	}
	return s.repo.Upsert(ctx, in)
}

func (s *Service) SetStatus(ctx context.Context, id, status string) (*Publication, error) {
	if !IsValidStatus(status) {
		return nil, ErrInvalidStatus
	}
	return s.repo.SetStatus(ctx, id, status)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}

type ListResult struct {
	Items      []Publication
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
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items, total, err := s.repo.List(ctx, ListFilters{
		SiteID:  p.SiteID,
		VideoID: p.VideoID,
		Status:  p.Status,
		Limit:   limit,
		Offset:  (page - 1) * limit,
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

func (s *Service) ListSiteDistribution(ctx context.Context, videoID string) ([]DistributionRow, error) {
	return s.repo.ListSiteDistribution(ctx, videoID)
}

func (s *Service) SaveSiteDistribution(ctx context.Context, videoID string, in SiteDistributionRequest) (*Publication, error) {
	if !IsValidStatus(in.Status) {
		return nil, ErrInvalidStatus
	}
	if !categorymappings.IsValidSyncMode(in.SyncMode) {
		return nil, ErrInvalidSyncMode
	}
	rows, err := s.repo.ListSiteDistribution(ctx, videoID)
	if err != nil {
		return nil, err
	}
	var target *DistributionRow
	for i := range rows {
		if rows[i].SiteID == in.SiteID {
			target = &rows[i]
			break
		}
	}
	if target == nil {
		return nil, ErrNotFound
	}
	if in.Status == StatusApproved && !distributionHasMappedCategory(target) {
		return nil, ErrMissingCategoryMapping
	}
	if in.SyncMode == categorymappings.SyncModePublish && target.MappingSyncMode == categorymappings.SyncModeBlock {
		return nil, ErrBlockedCategorySync
	}
	mode := in.SyncMode
	return s.repo.Upsert(ctx, UpsertInput{
		SiteID:   in.SiteID,
		VideoID:  videoID,
		Status:   in.Status,
		SyncMode: &mode,
	})
}

func distributionHasMappedCategory(row *DistributionRow) bool {
	for _, m := range row.CategoryMappings {
		if m.RemoteCategoryID != nil && m.MappingEnabled && m.MappingSyncMode != categorymappings.SyncModeBlock {
			return true
		}
	}
	return row.RemoteCategoryID != nil && row.MappingEnabled && row.MappingSyncMode != categorymappings.SyncModeBlock
}

type BulkDistributionParams struct {
	SiteID           string
	VideoIDs         []string
	Status           string
	SyncMode         string
	UnmappedBehavior string
}

func (s *Service) BulkSaveSiteDistribution(ctx context.Context, p BulkDistributionParams) (*BulkDistributionResponse, error) {
	if !isUUID(p.SiteID) {
		return nil, ErrNotFound
	}
	if !IsValidBulkStatus(p.Status) {
		return nil, ErrInvalidStatus
	}
	if !categorymappings.IsValidSyncMode(p.SyncMode) {
		return nil, ErrInvalidSyncMode
	}
	if p.UnmappedBehavior == "" {
		p.UnmappedBehavior = "skip"
	}
	if p.UnmappedBehavior != "skip" && p.UnmappedBehavior != "draft" {
		return nil, ErrInvalidUnmappedBehavior
	}
	seen := map[string]struct{}{}
	videoIDs := make([]string, 0, len(p.VideoIDs))
	for _, id := range p.VideoIDs {
		if !isUUID(id) {
			return nil, ErrInvalidVideoID
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		videoIDs = append(videoIDs, id)
	}
	return s.repo.BulkUpsertForSite(ctx, BulkUpsertInput{
		SiteID:           p.SiteID,
		VideoIDs:         videoIDs,
		Status:           p.Status,
		SyncMode:         p.SyncMode,
		UnmappedBehavior: p.UnmappedBehavior,
	})
}

func IsValidBulkStatus(status string) bool {
	switch status {
	case StatusDraft, StatusApproved, StatusBlocked:
		return true
	default:
		return false
	}
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool {
	return uuidPattern.MatchString(s)
}
