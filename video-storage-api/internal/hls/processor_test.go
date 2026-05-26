package hls

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMasterPlaylist_Multivariant_Landscape(t *testing.T) {
	rendered := []RenderedVariant{
		{Spec: DefaultVariants[0], OutputWidth: 640, OutputHeight: 360},
		{Spec: DefaultVariants[1], OutputWidth: 854, OutputHeight: 480},
		{Spec: DefaultVariants[2], OutputWidth: 1280, OutputHeight: 720},
	}
	got := BuildMasterPlaylist(rendered)

	for _, want := range []string{
		"#EXTM3U",
		"#EXT-X-VERSION:3",
		"#EXT-X-INDEPENDENT-SEGMENTS",
		"BANDWIDTH=900000,RESOLUTION=640x360",
		"360p/index.m3u8",
		"BANDWIDTH=1530000,RESOLUTION=854x480",
		"480p/index.m3u8",
		"BANDWIDTH=2630000,RESOLUTION=1280x720",
		"720p/index.m3u8",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("master playlist missing %q\nfull output:\n%s", want, got)
		}
	}
}

func TestBuildMasterPlaylist_SingleVariant(t *testing.T) {
	rendered := []RenderedVariant{
		{Spec: DefaultVariants[0], OutputWidth: 640, OutputHeight: 360},
	}
	got := BuildMasterPlaylist(rendered)

	if !strings.Contains(got, "360p/index.m3u8") {
		t.Errorf("expected 360p reference, got:\n%s", got)
	}
	if strings.Contains(got, "480p") || strings.Contains(got, "720p") {
		t.Errorf("single-variant master should not reference 480p/720p:\n%s", got)
	}
}

func TestBuildMasterPlaylist_Portrait(t *testing.T) {
	// 1080x1920 portrait → 720p output is 720x1280.
	w720, h720 := ComputeOutputDims(DefaultVariants[2], 1080, 1920)
	rendered := []RenderedVariant{
		{Spec: DefaultVariants[2], OutputWidth: w720, OutputHeight: h720},
	}
	got := BuildMasterPlaylist(rendered)
	if !strings.Contains(got, "RESOLUTION=720x1280") {
		t.Errorf("portrait master should advertise portrait dims, got:\n%s", got)
	}
}

// --- ValidateHLSOutput + CleanupFailedHLSJob ---

func newProbe(w, h int, ok bool) ResolutionProber {
	return staticProber{w: w, h: h, ok: ok}
}

type staticProber struct {
	w, h int
	ok   bool
}

func (s staticProber) ProbeResolution(_ context.Context, _ string) (int, int, bool) {
	return s.w, s.h, s.ok
}

func TestValidateHLSOutput_RejectsMissingFiles(t *testing.T) {
	tmp := t.TempDir()
	paths, err := NewPaths(tmp)
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	p := &Processor{paths: paths, prober: newProbe(1920, 1080, true), variants: DefaultVariants, cfg: DefaultProcessorConfig}

	vid := "9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b"

	// Nothing on disk — expect ErrMissingOutput.
	if err := p.ValidateHLSOutput(vid, []string{"720p"}); !errors.Is(err, ErrMissingOutput) {
		t.Fatalf("expected ErrMissingOutput, got %v", err)
	}

	// Write a master but no variant index.
	_ = os.MkdirAll(paths.VideoDir(vid), 0o750)
	_ = os.WriteFile(paths.MasterManifest(vid), []byte("#EXTM3U\n"), 0o640)
	if err := p.ValidateHLSOutput(vid, []string{"720p"}); !errors.Is(err, ErrMissingOutput) {
		t.Fatalf("expected ErrMissingOutput when variant index missing, got %v", err)
	}

	// Write the variant index too — now it should pass.
	_ = os.MkdirAll(paths.VariantDir(vid, "720p"), 0o750)
	_ = os.WriteFile(paths.VariantIndex(vid, "720p"), []byte("#EXTM3U\n#EXT-X-ENDLIST\n"), 0o640)
	if err := p.ValidateHLSOutput(vid, []string{"720p"}); err != nil {
		t.Fatalf("expected success after writing both, got %v", err)
	}
}

func TestValidateHLSOutput_RejectsEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	paths, err := NewPaths(tmp)
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	p := &Processor{paths: paths, prober: newProbe(1920, 1080, true), variants: DefaultVariants, cfg: DefaultProcessorConfig}

	vid := "9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b"
	_ = os.MkdirAll(paths.VariantDir(vid, "720p"), 0o750)
	// Empty master.
	_ = os.WriteFile(paths.MasterManifest(vid), nil, 0o640)
	_ = os.WriteFile(paths.VariantIndex(vid, "720p"), []byte("ok"), 0o640)
	if err := p.ValidateHLSOutput(vid, []string{"720p"}); !errors.Is(err, ErrMissingOutput) {
		t.Fatalf("expected ErrMissingOutput for empty file, got %v", err)
	}
}

func TestValidateHLSOutput_RejectsBadVariantName(t *testing.T) {
	tmp := t.TempDir()
	paths, err := NewPaths(tmp)
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	p := &Processor{paths: paths, prober: newProbe(1920, 1080, true), variants: DefaultVariants, cfg: DefaultProcessorConfig}
	vid := "9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b"
	if err := p.ValidateHLSOutput(vid, []string{"1080p"}); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath for un-whitelisted variant, got %v", err)
	}
}

func TestCleanupFailedHLSJob_RemovesEverything(t *testing.T) {
	tmp := t.TempDir()
	paths, err := NewPaths(tmp)
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	p := &Processor{paths: paths, prober: newProbe(0, 0, false), variants: DefaultVariants, cfg: DefaultProcessorConfig}
	vid := "9b1f5b8a-0d28-4d5e-9b71-bccaee0a3d7b"

	// Lay down some partial files.
	if err := os.MkdirAll(paths.VariantDir(vid, "360p"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"segment000.ts", "segment001.ts"} {
		if err := os.WriteFile(filepath.Join(paths.VariantDir(vid, "360p"), name), []byte("partial"), 0o640); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := os.WriteFile(paths.MasterManifest(vid), []byte("#EXTM3U\n"), 0o640); err != nil {
		t.Fatalf("master write: %v", err)
	}

	if err := p.CleanupFailedHLSJob(vid); err != nil {
		t.Fatalf("CleanupFailedHLSJob: %v", err)
	}

	if _, err := os.Stat(paths.VideoDir(vid)); !os.IsNotExist(err) {
		t.Fatalf("expected VideoDir removed; stat err = %v", err)
	}
}
