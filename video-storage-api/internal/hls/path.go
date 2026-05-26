package hls

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MasterFilename is the canonical top-level manifest name.
const MasterFilename = "master.m3u8"

// VariantIndexFilename is the per-variant playlist name.
const VariantIndexFilename = "index.m3u8"

// allowedVariants gates URL params before they reach filepath.Join.
// Adding a new variant requires whitelisting it here AND giving it a
// rung on DefaultVariants in variants.go.
var allowedVariants = map[string]struct{}{
	"360p": {},
	"480p": {},
	"720p": {},
}

var (
	// 8-4-4-4-12 hex, case-insensitive — matches Postgres' uuid output.
	videoIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	// segmentNNN.ts or segmentNNN.m4s (3-5 digits). ffmpeg uses %03d
	// by default but we allow extras for future longer videos. init.mp4
	// is accepted for fragmented-MP4 HLS compatibility.
	segmentRegex = regexp.MustCompile(`^(segment\d{3,5}\.(ts|m4s)|init\.mp4)$`)
)

// Paths bundles a private HLS root directory with its sub-path helpers.
// The root is created on construction; everything else is pure
// string-building plus validation.
type Paths struct {
	baseDir string // absolute
}

// NewPaths resolves baseDir to absolute, creates it (and any parents)
// with restrictive perms, and verifies it is a directory.
func NewPaths(baseDir string) (*Paths, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("hls: resolve base dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("hls: mkdir base dir: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("hls: stat base dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("hls: base path is not a directory: %s", abs)
	}
	return &Paths{baseDir: abs}, nil
}

// BaseDir returns the absolute root.
func (p *Paths) BaseDir() string { return p.baseDir }

// VideoDir returns the absolute directory for a single video's HLS output.
// Caller must have validated videoID first.
func (p *Paths) VideoDir(videoID string) string {
	return filepath.Join(p.baseDir, strings.ToLower(videoID))
}

// VariantDir returns the absolute directory for one variant.
func (p *Paths) VariantDir(videoID, variant string) string {
	return filepath.Join(p.VideoDir(videoID), variant)
}

// MasterManifest returns the absolute path to master.m3u8.
func (p *Paths) MasterManifest(videoID string) string {
	return filepath.Join(p.VideoDir(videoID), MasterFilename)
}

// VariantIndex returns the absolute path to a variant's playlist.
func (p *Paths) VariantIndex(videoID, variant string) string {
	return filepath.Join(p.VariantDir(videoID, variant), VariantIndexFilename)
}

// Segment returns the absolute path to a single HLS segment.
func (p *Paths) Segment(videoID, variant, segment string) string {
	return filepath.Join(p.VariantDir(videoID, variant), segment)
}

// ValidateVideoID rejects anything that isn't a canonical UUID. The
// resulting string is safe to pass to filepath.Join.
func ValidateVideoID(id string) error {
	if !videoIDRegex.MatchString(id) {
		return ErrInvalidPath
	}
	return nil
}

// ValidateVariant rejects any name not in the static whitelist.
func ValidateVariant(v string) error {
	if _, ok := allowedVariants[v]; !ok {
		return ErrInvalidPath
	}
	return nil
}

// ValidateSegment rejects anything that isn't `segmentNNN.ts`.
func ValidateSegment(s string) error {
	if !segmentRegex.MatchString(s) {
		return ErrInvalidPath
	}
	return nil
}

// IsVariantAllowed reports whether `v` is in the whitelist (handy for
// tests / config inspection without throwing an error).
func IsVariantAllowed(v string) bool {
	_, ok := allowedVariants[v]
	return ok
}
