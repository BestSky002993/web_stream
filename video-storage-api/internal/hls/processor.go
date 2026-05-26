package hls

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// ProcessorConfig controls the multi-variant HLS conversion.
type ProcessorConfig struct {
	// HLSSegmentSeconds is the target duration of each .ts segment.
	// 4s is the Phase 14 default — short enough for fast quality
	// switches under hls.js, long enough to keep request count down.
	HLSSegmentSeconds int

	// ConvertTimeout caps a single video's full conversion (all
	// selected variants combined). Multi-variant re-encode is slow on
	// commodity hardware, so this is generous by default.
	ConvertTimeout time.Duration

	// X264Preset selects the libx264 speed / quality tradeoff used for
	// every variant. `veryfast` is the Phase 14 default — production
	// queue workers (future phase) can drop to `medium` for better
	// bitrate efficiency at the cost of conversion time.
	X264Preset string
}

// DefaultProcessorConfig is the Phase 14 default ladder timing.
var DefaultProcessorConfig = ProcessorConfig{
	HLSSegmentSeconds: 4,
	ConvertTimeout:    60 * time.Minute,
	X264Preset:        "veryfast",
}

// ResolutionProber returns the source video's (width, height) in
// pixels. Implemented by storage.LocalStorage; abstracted to keep the
// HLS package testable without a real storage backend.
type ResolutionProber interface {
	ProbeResolution(ctx context.Context, videoFilename string) (int, int, bool)
}

// Processor runs ffmpeg HLS conversions against on-disk video files.
type Processor struct {
	paths          *Paths
	videoSourceDir string // absolute path where uploaded videos live
	prober         ResolutionProber
	variants       []VariantSpec
	cfg            ProcessorConfig
}

// NewProcessor returns a Processor wired to the HLS output root, the
// directory holding source MP4 files, and a resolution prober.
func NewProcessor(paths *Paths, videoSourceDir string, prober ResolutionProber, cfg ProcessorConfig) *Processor {
	if cfg.HLSSegmentSeconds <= 0 {
		cfg.HLSSegmentSeconds = DefaultProcessorConfig.HLSSegmentSeconds
	}
	if cfg.ConvertTimeout <= 0 {
		cfg.ConvertTimeout = DefaultProcessorConfig.ConvertTimeout
	}
	if cfg.X264Preset == "" {
		cfg.X264Preset = DefaultProcessorConfig.X264Preset
	}
	return &Processor{
		paths:          paths,
		videoSourceDir: videoSourceDir,
		prober:         prober,
		variants:       DefaultVariants,
		cfg:            cfg,
	}
}

// ConvertToHLS produces every eligible variant for the given video.
//
// Returns the list of variant names actually rendered (in master-
// playlist order). The caller persists this list so the dashboard can
// show "Qualities: 360p, 480p, 720p" without re-scanning the disk.
//
// On any failure mid-pipeline the entire video output directory is
// removed so a retry has a clean slate.
func (p *Processor) ConvertToHLS(ctx context.Context, videoID, sourceFilename string) ([]string, error) {
	if err := ValidateVideoID(videoID); err != nil {
		return nil, err
	}

	inputPath := filepath.Join(p.videoSourceDir, sourceFilename)
	if _, err := os.Stat(inputPath); err != nil {
		return nil, fmt.Errorf("hls: stat source: %w", err)
	}

	// Detect source resolution so we don't upscale. Probe failure is
	// non-fatal — SelectVariants falls back to the lowest rung.
	srcW, srcH, _ := p.prober.ProbeResolution(ctx, sourceFilename)
	eligible := SelectVariants(p.variants, srcW, srcH)
	if len(eligible) == 0 {
		return nil, fmt.Errorf("%w: no eligible variants", ErrMissingOutput)
	}

	if err := os.MkdirAll(p.paths.VideoDir(videoID), 0o750); err != nil {
		return nil, fmt.Errorf("hls: mkdir video dir: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(p.paths.VideoDir(videoID))
	}

	rendered := make([]RenderedVariant, 0, len(eligible))
	for _, spec := range eligible {
		variantDir := p.paths.VariantDir(videoID, spec.Name)
		if err := os.MkdirAll(variantDir, 0o750); err != nil {
			cleanup()
			return nil, fmt.Errorf("hls: mkdir variant %s: %w", spec.Name, err)
		}
		if err := p.runFFmpegVariant(ctx, inputPath, variantDir, spec, srcW, srcH); err != nil {
			cleanup()
			return nil, err
		}
		outW, outH := ComputeOutputDims(spec, srcW, srcH)
		rendered = append(rendered, RenderedVariant{
			Spec:         spec,
			OutputWidth:  outW,
			OutputHeight: outH,
		})
	}

	if err := p.writeMasterManifest(videoID, rendered); err != nil {
		cleanup()
		return nil, err
	}

	names := namesOf(rendered)
	if err := p.ValidateHLSOutput(videoID, names); err != nil {
		cleanup()
		return nil, err
	}
	return names, nil
}

// runFFmpegVariant runs a single re-encode for one rung on the ladder.
// Aspect ratio is preserved via the variant's `scale` filter; bitrate,
// audio, and keyframe interval are set from the spec.
func (p *Processor) runFFmpegVariant(parent context.Context, inputPath, variantDir string, spec VariantSpec, srcW, srcH int) error {
	ctx, cancel := context.WithTimeout(parent, p.cfg.ConvertTimeout)
	defer cancel()

	segmentPattern := filepath.Join(variantDir, "segment%03d.ts")
	indexPath := filepath.Join(variantDir, VariantIndexFilename)

	// Keyframe every <segment>*30 frames (assume up to 30 fps source)
	// so HLS segment boundaries fall on keyframes. -sc_threshold 0
	// disables scene-cut keyframes that would misalign segments.
	gop := strconv.Itoa(p.cfg.HLSSegmentSeconds * 30)

	// Every argument is a discrete element — no shell, no string
	// interpolation, so user input can't smuggle extra flags.
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-loglevel", "error",
		"-y",
		"-i", inputPath,
		"-vf", spec.ScaleFilter(srcW, srcH),
		"-c:v", "libx264",
		"-preset", p.cfg.X264Preset,
		"-profile:v", "main",
		"-level", "4.0",
		"-pix_fmt", "yuv420p",
		"-b:v", spec.VideoBitrate,
		"-maxrate", spec.VideoMaxrate,
		"-bufsize", spec.VideoBufsize,
		"-g", gop,
		"-keyint_min", gop,
		"-sc_threshold", "0",
		"-c:a", "aac",
		"-b:a", spec.AudioBitrate,
		"-ar", "48000",
		"-ac", "2",
		"-hls_time", strconv.Itoa(p.cfg.HLSSegmentSeconds),
		"-hls_list_size", "0",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", segmentPattern,
		"-f", "hls",
		indexPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w (%s): %v (stderr=%q)", ErrFFmpegFailed, spec.Name, err, truncate(stderr.String(), 500))
	}
	return nil
}

// writeMasterManifest emits master.m3u8 referencing every rendered
// variant, ordered by ascending bandwidth (smallest first).
func (p *Processor) writeMasterManifest(videoID string, rendered []RenderedVariant) error {
	path := p.paths.MasterManifest(videoID)
	return os.WriteFile(path, []byte(BuildMasterPlaylist(rendered)), 0o640)
}

// BuildMasterPlaylist is the pure renderer for master.m3u8 contents.
// Exported so tests can verify the output byte-for-byte.
func BuildMasterPlaylist(rendered []RenderedVariant) string {
	var b bytes.Buffer
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")
	b.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	for _, r := range rendered {
		fmt.Fprintf(&b,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"avc1.4d401f,mp4a.40.2\"\n",
			r.Spec.BandwidthBPS, r.OutputWidth, r.OutputHeight,
		)
		fmt.Fprintf(&b, "%s/%s\n", r.Spec.Name, VariantIndexFilename)
	}
	return b.String()
}

// CleanupFailedHLSJob removes any partially-written output. Safe to
// call on success too (it then removes the good output — only call on
// failure paths).
func (p *Processor) CleanupFailedHLSJob(videoID string) error {
	if err := ValidateVideoID(videoID); err != nil {
		return err
	}
	return os.RemoveAll(p.paths.VideoDir(videoID))
}

// ValidateHLSOutput verifies the expected master + variant files are
// on disk and non-empty for each provided variant name.
func (p *Processor) ValidateHLSOutput(videoID string, variants []string) error {
	if err := ValidateVideoID(videoID); err != nil {
		return err
	}
	checks := []string{p.paths.MasterManifest(videoID)}
	for _, v := range variants {
		if err := ValidateVariant(v); err != nil {
			return err
		}
		checks = append(checks, p.paths.VariantIndex(videoID, v))
	}
	for _, path := range checks {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrMissingOutput, path)
		}
		if info.Size() == 0 {
			return fmt.Errorf("%w: %s (empty)", ErrMissingOutput, path)
		}
	}
	return nil
}

func namesOf(rs []RenderedVariant) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Spec.Name
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
