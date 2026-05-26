package categorymappings

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListRemoteCategories(ctx context.Context, siteID string) ([]RemoteCategory, error) {
	return s.repo.ListRemoteCategories(ctx, siteID)
}

func (s *Service) ImportRemoteCategories(ctx context.Context, siteID string, items []RemoteCategoryImport) ([]RemoteCategory, error) {
	cleaned := make([]RemoteCategoryImport, 0, len(items))
	for _, item := range items {
		v, err := NormalizeRemoteCategory(item)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, v)
	}
	return s.repo.ImportRemoteCategories(ctx, siteID, cleaned)
}

func (s *Service) ListMappings(ctx context.Context, siteID string) ([]MappingRow, error) {
	return s.repo.ListMappings(ctx, siteID)
}

func (s *Service) UpsertAdminMappings(ctx context.Context, siteID string, items []MappingUpsertInput) ([]MappingRow, error) {
	for i := range items {
		if !IsValidSyncMode(items[i].SyncMode) {
			return nil, ErrInvalidMode
		}
	}
	if err := s.repo.UpsertAdminMappings(ctx, siteID, items); err != nil {
		return nil, err
	}
	return s.repo.ListMappings(ctx, siteID)
}

func (s *Service) UpdateSiteMapping(ctx context.Context, siteID, mappingID string, in SiteMappingUpdateInput) (*MappingRow, error) {
	if !IsValidSyncMode(in.SyncMode) {
		return nil, ErrInvalidMode
	}
	return s.repo.UpdateSiteMapping(ctx, siteID, mappingID, in)
}

func (s *Service) SuggestSameNameMappings(ctx context.Context, siteID string) ([]MappingRow, error) {
	if err := s.repo.SuggestSameNameMappings(ctx, siteID); err != nil {
		return nil, err
	}
	return s.repo.ListMappings(ctx, siteID)
}

func (s *Service) MappingStatusForCategories(ctx context.Context, siteID string, categoryIDs []string) (map[string]MappingRow, error) {
	return s.repo.MappingStatusForCategories(ctx, siteID, categoryIDs)
}
