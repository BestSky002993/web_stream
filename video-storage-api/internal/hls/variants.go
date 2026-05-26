package hls

// VariantSpec is one rung on the adaptive bitrate ladder.
//
// `TargetShortSide` is the dimension the ffmpeg `scale` filter clamps:
// the short side (height for landscape, width for portrait) becomes
// this value, and the long side scales proportionally to a
// codec-friendly even number via the `-2` modifier.
//
// Bitrates are conservative defaults aimed at typical web playback.
// Multi-pass / VBR would do better quality but cost more CPU; these
// are tuned for veryfast preset on commodity hardware.
type VariantSpec struct {
	Name            string // "360p", "480p", "720p"
	TargetShortSide int    // 360 / 480 / 720
	VideoBitrate    string // ffmpeg -b:v (e.g. "800k")
	VideoMaxrate    string // ffmpeg -maxrate
	VideoBufsize    string // ffmpeg -bufsize
	AudioBitrate    string // ffmpeg -b:a
	BandwidthBPS    int    // master.m3u8 BANDWIDTH attribute (total bits/sec)
}

// DefaultVariants is the ladder Phase 14 ships. 1080p is intentionally
// excluded — see the README for why.
var DefaultVariants = []VariantSpec{
	{
		Name:            "360p",
		TargetShortSide: 360,
		VideoBitrate:    "800k",
		VideoMaxrate:    "856k",
		VideoBufsize:    "1200k",
		AudioBitrate:    "96k",
		BandwidthBPS:    900_000, // 800k video + 96k audio + container overhead
	},
	{
		Name:            "480p",
		TargetShortSide: 480,
		VideoBitrate:    "1400k",
		VideoMaxrate:    "1498k",
		VideoBufsize:    "2100k",
		AudioBitrate:    "128k",
		BandwidthBPS:    1_530_000,
	},
	{
		Name:            "720p",
		TargetShortSide: 720,
		VideoBitrate:    "2500k",
		VideoMaxrate:    "2675k",
		VideoBufsize:    "3750k",
		AudioBitrate:    "128k",
		BandwidthBPS:    2_630_000,
	},
}

// IsLandscape reports whether (w, h) is landscape or square (square
// counts as landscape — the scale filter will cap height).
func IsLandscape(w, h int) bool { return w >= h }

// SelectVariants returns the subset of `all` that can be generated
// from a source of the given dimensions without upscaling.
//
// A variant is eligible iff its TargetShortSide is <= min(w, h).
// Equal source/target dimensions are allowed (same-res re-encode is
// not an upscale).
//
// If `width` and `height` are both zero (probe failed), this returns
// the lowest single rung as a safe fallback so the pipeline still
// produces SOMETHING playable.
func SelectVariants(all []VariantSpec, width, height int) []VariantSpec {
	if width <= 0 || height <= 0 {
		// Unknown source resolution — generate only the safest rung.
		if len(all) > 0 {
			return []VariantSpec{all[0]}
		}
		return nil
	}
	short := width
	if height < short {
		short = height
	}
	out := make([]VariantSpec, 0, len(all))
	for _, v := range all {
		if v.TargetShortSide <= short {
			out = append(out, v)
		}
	}
	// If nothing qualified (source smaller than the smallest rung),
	// still emit the smallest rung — we accept a tiny upscale rather
	// than no HLS at all.
	if len(out) == 0 && len(all) > 0 {
		out = append(out, all[0])
	}
	return out
}

// RenderedVariant tracks the actual output dimensions ffmpeg will
// produce for a given source + spec. Used to populate the RESOLUTION
// attribute on master.m3u8 lines.
type RenderedVariant struct {
	Spec         VariantSpec
	OutputWidth  int
	OutputHeight int
}

// ComputeOutputDims figures out the (width, height) ffmpeg will emit
// after applying the variant's scale filter, given the source size.
// The long side is rounded to the nearest even integer so codecs that
// require chroma-divisible dimensions don't complain.
func ComputeOutputDims(spec VariantSpec, srcW, srcH int) (int, int) {
	if srcW <= 0 || srcH <= 0 {
		// Unknown source: assume 16:9 landscape for the metadata hint.
		if IsLandscape(16, 9) {
			return spec.TargetShortSide * 16 / 9, spec.TargetShortSide
		}
	}
	if IsLandscape(srcW, srcH) {
		// height = target, width auto.
		h := spec.TargetShortSide
		w := roundToEven(srcW * h / srcH)
		return w, h
	}
	// Portrait: width = target, height auto.
	w := spec.TargetShortSide
	h := roundToEven(srcH * w / srcW)
	return w, h
}

// ScaleFilter returns the ffmpeg `-vf scale=…` expression for the
// variant + source orientation. Aspect ratio is preserved; the long
// side rounds to an even integer via the `-2` modifier.
func (s VariantSpec) ScaleFilter(srcW, srcH int) string {
	if IsLandscape(srcW, srcH) {
		// scale=-2:H → height H, width auto-rounded to even.
		return "scale=-2:" + itoa(s.TargetShortSide)
	}
	// Portrait: scale=W:-2 → width W, height auto-rounded to even.
	return "scale=" + itoa(s.TargetShortSide) + ":-2"
}

func roundToEven(n int) int {
	if n < 2 {
		return 2
	}
	if n%2 == 0 {
		return n
	}
	return n + 1
}

// tiny itoa avoiding fmt for hot paths (and to keep this file free of
// the fmt import noise when the test file uses the helper directly).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
