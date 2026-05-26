package categorymappings

import "testing"

func TestNormalizeRemoteCategoryValidation(t *testing.T) {
	valid, err := NormalizeRemoteCategory(RemoteCategoryImport{
		RemoteTermID:   " 42 ",
		RemoteSlug:     " Clip-A ",
		RemoteName:     " คลิปเอ ",
		RemoteTaxonomy: " Category ",
	})
	if err != nil {
		t.Fatalf("NormalizeRemoteCategory returned error: %v", err)
	}
	if valid.RemoteSlug != "clip-a" || valid.RemoteTaxonomy != "category" || valid.RemoteName != "คลิปเอ" {
		t.Fatalf("unexpected normalized category: %#v", valid)
	}

	invalid := []RemoteCategoryImport{
		{RemoteTermID: "1", RemoteSlug: "", RemoteName: "A", RemoteTaxonomy: "category"},
		{RemoteTermID: "1", RemoteSlug: "../bad", RemoteName: "A", RemoteTaxonomy: "category"},
		{RemoteTermID: "1", RemoteSlug: "https://bad", RemoteName: "A", RemoteTaxonomy: "category"},
		{RemoteTermID: "1", RemoteSlug: "ok", RemoteName: "", RemoteTaxonomy: "category"},
	}
	for _, tc := range invalid {
		if _, err := NormalizeRemoteCategory(tc); err == nil {
			t.Fatalf("expected validation error for %#v", tc)
		}
	}
}

func TestNormalizeRemoteCategorySupportsWordPressTaxonomies(t *testing.T) {
	for _, taxonomy := range []string{"category", "video_category"} {
		got, err := NormalizeRemoteCategory(RemoteCategoryImport{
			RemoteTermID:   "1",
			RemoteSlug:     "clip-a",
			RemoteName:     "Clip A",
			RemoteTaxonomy: taxonomy,
		})
		if err != nil {
			t.Fatalf("NormalizeRemoteCategory(%s): %v", taxonomy, err)
		}
		if got.RemoteTaxonomy != taxonomy {
			t.Fatalf("taxonomy = %q, want %q", got.RemoteTaxonomy, taxonomy)
		}
	}
}

func TestNormalizeRemoteCategorySupportsThaiUTF8NameAndSlug(t *testing.T) {
	got, err := NormalizeRemoteCategory(RemoteCategoryImport{
		RemoteTermID:   "9",
		RemoteSlug:     "คลิปใหม่",
		RemoteName:     "คลิปใหม่",
		RemoteTaxonomy: "category",
	})
	if err != nil {
		t.Fatalf("NormalizeRemoteCategory returned error: %v", err)
	}
	if got.RemoteSlug != "คลิปใหม่" || got.RemoteName != "คลิปใหม่" {
		t.Fatalf("unexpected Thai normalization: %#v", got)
	}
}

func TestNormalizeRemoteCategoryDecodesURLEncodedSlug(t *testing.T) {
	got, err := NormalizeRemoteCategory(RemoteCategoryImport{
		RemoteTermID:   "10",
		RemoteSlug:     "%E0%B8%84%E0%B8%A5%E0%B8%B4%E0%B8%9B",
		RemoteName:     "คลิป",
		RemoteTaxonomy: "category",
	})
	if err != nil {
		t.Fatalf("NormalizeRemoteCategory returned error: %v", err)
	}
	if got.RemoteSlug != "คลิป" {
		t.Fatalf("remote slug = %q, want decoded Thai slug", got.RemoteSlug)
	}
}

func TestNormalizeRemoteCategoryValidationErrorIncludesField(t *testing.T) {
	_, err := NormalizeRemoteCategory(RemoteCategoryImport{RemoteTermID: "1", RemoteSlug: "", RemoteName: "A", RemoteTaxonomy: "category"})
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T %v", err, err)
	}
	if validationErr.Field != "remote_slug" || validationErr.Reason != "empty after sanitization" {
		t.Fatalf("unexpected validation error: %#v", validationErr)
	}
}

func TestSameNameSuggestionRowsAreDisabledByDefault(t *testing.T) {
	row := MappingRow{
		BackendCategory: BackendCategory{ID: "backend-x", Name: "Anime", Slug: "anime"},
		Mapping: &CategoryMapping{
			ID:                "mapping-1",
			BackendCategoryID: "backend-x",
			RemoteCategoryID:  strPtr("remote-a"),
			Enabled:           false,
			SyncMode:          SyncModeBlock,
			ConfirmedByWP:     false,
			ConfirmedByStream: false,
		},
		RemoteCategory: &RemoteCategory{ID: "remote-a", RemoteName: "Anime"},
	}
	row.Status = MappingStatus(row.Mapping)
	if row.Mapping.Enabled || row.Mapping.SyncMode != SyncModeBlock || row.Status != MappingStatusDisabled {
		t.Fatalf("suggested mapping must be disabled/block by default: %#v", row)
	}
}

func TestMappingStatusRules(t *testing.T) {
	if got := MappingStatus(nil); got != MappingStatusUnmapped {
		t.Fatalf("nil mapping status = %s", got)
	}
	remoteID := "remote-a"
	if got := MappingStatus(&CategoryMapping{RemoteCategoryID: &remoteID, Enabled: false}); got != MappingStatusDisabled {
		t.Fatalf("disabled mapping status = %s", got)
	}
	if got := MappingStatus(&CategoryMapping{RemoteCategoryID: &remoteID, Enabled: true, ConfirmedByStream: true}); got != MappingStatusUnconfirmed {
		t.Fatalf("half-confirmed mapping status = %s", got)
	}
	if got := MappingStatus(&CategoryMapping{RemoteCategoryID: &remoteID, Enabled: true, ConfirmedByStream: true, ConfirmedByWP: true}); got != MappingStatusMapped {
		t.Fatalf("confirmed mapping status = %s", got)
	}
}

func TestManyBackendCategoriesCanShareOneRemoteCategory(t *testing.T) {
	remoteID := "remote-a"
	rows := []MappingRow{
		{BackendCategory: BackendCategory{ID: "x"}, Mapping: &CategoryMapping{BackendCategoryID: "x", RemoteCategoryID: &remoteID, Enabled: true, ConfirmedByWP: true, ConfirmedByStream: true}},
		{BackendCategory: BackendCategory{ID: "y"}, Mapping: &CategoryMapping{BackendCategoryID: "y", RemoteCategoryID: &remoteID, Enabled: true, ConfirmedByWP: true, ConfirmedByStream: true}},
	}
	for _, row := range rows {
		if row.Mapping.RemoteCategoryID == nil || *row.Mapping.RemoteCategoryID != remoteID {
			t.Fatalf("backend category %s did not map to shared remote category", row.BackendCategory.ID)
		}
	}
}

func strPtr(s string) *string { return &s }
