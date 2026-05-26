package hls

import "errors"

// HLS pipeline statuses (mirrors the videos.hls_status CHECK).
const (
	StatusPending    = "pending"    // freshly inserted; conversion not yet started
	StatusProcessing = "processing" // ffmpeg is running
	StatusReady      = "ready"      // master.m3u8 + segments on disk
	StatusFailed     = "failed"     // ffmpeg or post-processing errored
	StatusSkipped    = "skipped"    // admin opted out / unsupported source
)

var (
	// ErrInvalidPath is returned for any URL parameter that doesn't pass
	// the path whitelist (bad UUID, unknown variant, bad segment name).
	// Never let user input reach filepath.Join without going through
	// the validators in path.go.
	ErrInvalidPath = errors.New("invalid hls path")

	// ErrNotReady is returned by the URL issuer when the video's
	// hls_status is not 'ready'. The handler maps this to 404 so the
	// caller can't probe pipeline state.
	ErrNotReady = errors.New("hls not ready for this video")

	// ErrNotFound is returned for unknown video IDs at the issuer.
	ErrNotFound = errors.New("video not found")

	// ErrMissingOutput is returned by the processor when ffmpeg exits
	// 0 but doesn't produce the expected files.
	ErrMissingOutput = errors.New("hls output missing after ffmpeg")

	// ErrFFmpegFailed wraps non-zero ffmpeg exits during HLS conversion.
	ErrFFmpegFailed = errors.New("hls ffmpeg conversion failed")
)
