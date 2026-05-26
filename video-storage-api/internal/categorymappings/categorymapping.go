package categorymappings

import "time"

const (
	SyncModePublish = "publish"
	SyncModeDraft   = "draft"
	SyncModeBlock   = "block"

	MappingStatusMapped      = "mapped"
	MappingStatusUnmapped    = "unmapped"
	MappingStatusDisabled    = "disabled"
	MappingStatusUnconfirmed = "unconfirmed"
)

type RemoteCategory struct {
	ID                 string
	SiteID             string
	RemoteTermID       string
	RemoteSlug         string
	RemoteName         string
	RemoteTaxonomy     string
	ParentRemoteTermID *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type BackendCategory struct {
	ID   string
	Name string
	Slug string
}

type CategoryMapping struct {
	ID                string
	SiteID            string
	BackendCategoryID string
	RemoteCategoryID  *string
	Enabled           bool
	SyncMode          string
	CreatedBy         *string
	ConfirmedByWP     bool
	ConfirmedByStream bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type MappingRow struct {
	BackendCategory BackendCategory
	Mapping         *CategoryMapping
	RemoteCategory  *RemoteCategory
	Status          string
}

type RemoteCategoryImport struct {
	RemoteTermID       string
	RemoteSlug         string
	RemoteName         string
	RemoteTaxonomy     string
	ParentRemoteTermID *string
}

type MappingUpsertInput struct {
	BackendCategoryID string
	RemoteCategoryID  *string
	Enabled           bool
	SyncMode          string
	ConfirmedByStream bool
}

type SiteMappingUpdateInput struct {
	RemoteCategoryID *string
	Enabled          bool
	SyncMode         string
	ConfirmedByWP    bool
}

func IsValidSyncMode(mode string) bool {
	switch mode {
	case SyncModePublish, SyncModeDraft, SyncModeBlock:
		return true
	default:
		return false
	}
}

func MappingStatus(m *CategoryMapping) string {
	if m == nil || m.RemoteCategoryID == nil {
		return MappingStatusUnmapped
	}
	if !m.Enabled {
		return MappingStatusDisabled
	}
	if !m.ConfirmedByStream || !m.ConfirmedByWP {
		return MappingStatusUnconfirmed
	}
	return MappingStatusMapped
}
