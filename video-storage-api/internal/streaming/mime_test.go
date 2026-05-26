package streaming

import "testing"

func TestContentTypeForExt(t *testing.T) {
	cases := map[string]string{
		"mp4":  "video/mp4",
		"MP4":  "video/mp4",
		".mp4": "video/mp4",
		"webm": "video/webm",
		"mov":  "video/quicktime",
		"xyz":  "application/octet-stream",
		"":     "application/octet-stream",
	}
	for in, want := range cases {
		got := contentTypeForExt(in)
		if got != want {
			t.Errorf("contentTypeForExt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContentTypeForThumbnail(t *testing.T) {
	cases := map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
		"gif":  "application/octet-stream",
	}
	for in, want := range cases {
		got := contentTypeForThumbnail(in)
		if got != want {
			t.Errorf("contentTypeForThumbnail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtOf(t *testing.T) {
	cases := map[string]string{
		"abc.mp4":     "mp4",
		"abc.def.mov": "mov",
		"noext":       "",
		"trailing.":   "",
		".hidden":     "hidden",
	}
	for in, want := range cases {
		got := extOf(in)
		if got != want {
			t.Errorf("extOf(%q) = %q, want %q", in, got, want)
		}
	}
}
