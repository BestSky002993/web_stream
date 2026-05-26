package categorymappings

import (
	"encoding/json"
	"testing"
)

func TestRemoteCategoryImportRequestCategoriesRoot(t *testing.T) {
	body := []byte(`{
		"categories": [{
			"remote_term_id": "1",
			"remote_slug": "uncategorized",
			"remote_name": "Uncategorized",
			"remote_taxonomy": "category",
			"parent_remote_term_id": null
		}]
	}`)
	var req RemoteCategoryImportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	items := req.items()
	if len(items) != 1 {
		t.Fatalf("expected 1 category, got %d", len(items))
	}
	if items[0].ParentRemoteTermID != nil {
		t.Fatalf("expected null parent_remote_term_id to decode as nil")
	}
	if items[0].RemoteTaxonomy != "category" {
		t.Fatalf("unexpected taxonomy %q", items[0].RemoteTaxonomy)
	}
}

func TestRemoteCategoryImportRequestMultipleCategories(t *testing.T) {
	body := []byte(`{
		"categories": [
			{"remote_term_id":"1","remote_slug":"a","remote_name":"A","remote_taxonomy":"category","parent_remote_term_id":""},
			{"remote_term_id":"2","remote_slug":"b","remote_name":"B","remote_taxonomy":"video_category","parent_remote_term_id":"1"}
		]
	}`)
	var req RemoteCategoryImportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	items := importItems(req.items())
	if len(items) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(items))
	}
	first, err := NormalizeRemoteCategory(items[0])
	if err != nil {
		t.Fatalf("NormalizeRemoteCategory first: %v", err)
	}
	if first.ParentRemoteTermID != nil {
		t.Fatalf("expected empty parent_remote_term_id to normalize to nil")
	}
	second, err := NormalizeRemoteCategory(items[1])
	if err != nil {
		t.Fatalf("NormalizeRemoteCategory second: %v", err)
	}
	if second.RemoteTaxonomy != "video_category" {
		t.Fatalf("unexpected taxonomy %q", second.RemoteTaxonomy)
	}
	if second.ParentRemoteTermID == nil || *second.ParentRemoteTermID != "1" {
		t.Fatalf("expected string parent term id to survive normalization")
	}
}
