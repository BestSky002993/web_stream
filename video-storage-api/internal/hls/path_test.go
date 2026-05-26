package hls

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateVideoID(t *testing.T) {
	good := []string{
		"9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b",
		"AABBCCDD-EEFF-0011-2233-445566778899", // upper-case allowed
	}
	for _, id := range good {
		if err := ValidateVideoID(id); err != nil {
			t.Errorf("expected %q to be valid: %v", id, err)
		}
	}

	bad := []string{
		"",
		"not-a-uuid",
		"9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7", // 11 chars in last group
		"9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7bb",
		"../etc/passwd",
		"9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b/extra",
	}
	for _, id := range bad {
		if err := ValidateVideoID(id); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("expected %q rejected, got %v", id, err)
		}
	}
}

func TestValidateVariant(t *testing.T) {
	for _, v := range []string{"360p", "480p", "720p"} {
		if err := ValidateVariant(v); err != nil {
			t.Errorf("expected %q valid: %v", v, err)
		}
	}
	for _, v := range []string{
		"", "..", "../720p", "720p/extra",
		"1080p",            // intentionally not in the ladder
		"V1", "v1", "720P", // case mismatches
		"360p/",            // trailing slash
	} {
		if err := ValidateVariant(v); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("expected %q rejected, got %v", v, err)
		}
	}
}

func TestValidateSegment(t *testing.T) {
	good := []string{"segment000.ts", "segment001.ts", "segment9999.ts", "segment12345.ts", "segment000.m4s", "init.mp4"}
	for _, s := range good {
		if err := ValidateSegment(s); err != nil {
			t.Errorf("expected %q valid: %v", s, err)
		}
	}
	bad := []string{
		"",
		"segment.ts",                // no digits
		"segment0.ts",               // <3 digits
		"segment123456.ts",          // >5 digits
		"segment000.tsx",
		"segment000.mp4",
		"init.m4s",
		"index.m3u8",
		"../segment000.ts",
		"segment000.ts/extra",
		"SEGMENT000.TS",
	}
	for _, s := range bad {
		if err := ValidateSegment(s); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("expected %q rejected, got %v", s, err)
		}
	}
}

func TestPaths_AreSubdirectoriesOfBase(t *testing.T) {
	tmp := t.TempDir()
	p, err := NewPaths(tmp)
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	vid := "9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b"
	got := []string{
		p.VideoDir(vid),
		p.VariantDir(vid, "720p"),
		p.MasterManifest(vid),
		p.VariantIndex(vid, "720p"),
		p.Segment(vid, "720p", "segment000.ts"),
	}
	for _, path := range got {
		if !strings.HasPrefix(path, p.BaseDir()) {
			t.Errorf("path %q is not under BaseDir %q", path, p.BaseDir())
		}
	}
}
