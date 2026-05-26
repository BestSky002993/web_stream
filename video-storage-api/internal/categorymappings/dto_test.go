package categorymappings

import "testing"

func TestToMappingResponseIncludesConfirmationAliases(t *testing.T) {
	remoteID := "remote-category-id"
	row := MappingRow{
		BackendCategory: BackendCategory{ID: "backend-category-id", Name: "คลิปเอ", Slug: "clip-a"},
		Mapping: &CategoryMapping{
			ID:                "mapping-id",
			BackendCategoryID: "backend-category-id",
			RemoteCategoryID:  &remoteID,
			Enabled:           true,
			SyncMode:          SyncModePublish,
			ConfirmedByWP:     true,
			ConfirmedByStream: true,
		},
		Status: MappingStatusMapped,
	}

	resp := ToMappingResponse(row)
	if !resp.ConfirmedByWP || !resp.WPConfirmed {
		t.Fatalf("expected WordPress confirmation fields to be true: %#v", resp)
	}
	if !resp.ConfirmedByStream || !resp.StreamConfirmed {
		t.Fatalf("expected stream confirmation fields to be true: %#v", resp)
	}
	if !resp.MappingConfirmed {
		t.Fatalf("expected mapping_confirmed to be true: %#v", resp)
	}
}
