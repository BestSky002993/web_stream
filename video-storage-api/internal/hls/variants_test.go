package hls

import (
	"reflect"
	"testing"
)

func TestIsLandscape(t *testing.T) {
	cases := []struct {
		w, h int
		want bool
	}{
		{1920, 1080, true},
		{1280, 720, true},
		{1080, 1920, false},
		{720, 1280, false},
		{600, 600, true}, // square counts as landscape
		{1, 1, true},
	}
	for _, c := range cases {
		if got := IsLandscape(c.w, c.h); got != c.want {
			t.Errorf("IsLandscape(%d, %d) = %v, want %v", c.w, c.h, got, c.want)
		}
	}
}

func TestSelectVariants_Landscape(t *testing.T) {
	cases := []struct {
		name string
		w, h int
		want []string
	}{
		{"1080p source",          1920, 1080, []string{"360p", "480p", "720p"}},
		{"720p source",           1280, 720,  []string{"360p", "480p", "720p"}},
		{"720p exact short side", 9999, 720,  []string{"360p", "480p", "720p"}},
		{"480p source",           854, 480,   []string{"360p", "480p"}},
		{"360p source",           640, 360,   []string{"360p"}},
		{"below 360p",            320, 240,   []string{"360p"}}, // single fallback
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := names(SelectVariants(DefaultVariants, c.w, c.h))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("SelectVariants(%dx%d) = %v, want %v", c.w, c.h, got, c.want)
			}
		})
	}
}

func TestSelectVariants_PortraitSameSelection(t *testing.T) {
	// Portrait videos use the short side just like landscape — a
	// 1080x1920 portrait has short side 1080, so all rungs qualify.
	got := names(SelectVariants(DefaultVariants, 1080, 1920))
	want := []string{"360p", "480p", "720p"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("portrait 1080p selection = %v, want %v", got, want)
	}

	// 480x854 portrait — short side 480, so 360p + 480p only.
	got = names(SelectVariants(DefaultVariants, 480, 854))
	want = []string{"360p", "480p"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("portrait 480p selection = %v, want %v", got, want)
	}
}

func TestSelectVariants_UnknownResolutionFallsBackToLowest(t *testing.T) {
	got := names(SelectVariants(DefaultVariants, 0, 0))
	if !reflect.DeepEqual(got, []string{"360p"}) {
		t.Errorf("unknown res → %v, want [360p]", got)
	}
}

func TestSelectVariants_NeverUpscales(t *testing.T) {
	// 850x478 source (just under 480p). Only 360p should make the cut.
	got := names(SelectVariants(DefaultVariants, 850, 478))
	if !reflect.DeepEqual(got, []string{"360p"}) {
		t.Errorf("478p source → %v, want [360p]", got)
	}
}

func TestComputeOutputDims_Landscape(t *testing.T) {
	cases := []struct {
		name      string
		spec      VariantSpec
		srcW, srcH int
		wantW, wantH int
	}{
		{"1080p → 720p", DefaultVariants[2], 1920, 1080, 1280, 720},
		{"1080p → 480p", DefaultVariants[1], 1920, 1080, 854, 480}, // 1920*480/1080 = 853.33 → 854 even
		{"1080p → 360p", DefaultVariants[0], 1920, 1080, 640, 360},
		// 5:4 source — non-16:9 — RESOLUTION reflects the real aspect.
		{"1280x1024 → 720p", DefaultVariants[2], 1280, 1024, 900, 720}, // 1280*720/1024 = 900
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, h := ComputeOutputDims(c.spec, c.srcW, c.srcH)
			if w != c.wantW || h != c.wantH {
				t.Errorf("ComputeOutputDims = %dx%d, want %dx%d", w, h, c.wantW, c.wantH)
			}
		})
	}
}

func TestComputeOutputDims_Portrait(t *testing.T) {
	cases := []struct {
		name       string
		spec       VariantSpec
		srcW, srcH int
		wantW      int
		wantH      int
	}{
		{"1080x1920 → 720p", DefaultVariants[2], 1080, 1920, 720, 1280},
		{"1080x1920 → 480p", DefaultVariants[1], 1080, 1920, 480, 854}, // 1920*480/1080 = 853.33 → 854
		{"1080x1920 → 360p", DefaultVariants[0], 1080, 1920, 360, 640},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, h := ComputeOutputDims(c.spec, c.srcW, c.srcH)
			if w != c.wantW || h != c.wantH {
				t.Errorf("ComputeOutputDims = %dx%d, want %dx%d", w, h, c.wantW, c.wantH)
			}
		})
	}
}

func TestScaleFilter(t *testing.T) {
	v := DefaultVariants[2] // 720p
	if got := v.ScaleFilter(1920, 1080); got != "scale=-2:720" {
		t.Errorf("landscape: got %q, want scale=-2:720", got)
	}
	if got := v.ScaleFilter(1080, 1920); got != "scale=720:-2" {
		t.Errorf("portrait: got %q, want scale=720:-2", got)
	}
	if got := v.ScaleFilter(600, 600); got != "scale=-2:720" {
		t.Errorf("square should pick landscape filter: got %q", got)
	}
}

func TestRoundToEven(t *testing.T) {
	cases := map[int]int{0: 2, 1: 2, 2: 2, 3: 4, 853: 854, 854: 854}
	for in, want := range cases {
		if got := roundToEven(in); got != want {
			t.Errorf("roundToEven(%d) = %d, want %d", in, got, want)
		}
	}
}

func names(specs []VariantSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}
