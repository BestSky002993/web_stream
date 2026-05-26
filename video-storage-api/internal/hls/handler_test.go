package hls

import (
	"strings"
	"testing"
)

func TestRewriteManifestURIs_MasterPlaylist(t *testing.T) {
	input := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-STREAM-INF:BANDWIDTH=2500000,CODECS="avc1.4d401f,mp4a.40.2"
v1/index.m3u8
`
	out := string(rewriteManifestURIs([]byte(input), "expires=123&signature=abc"))
	if !strings.Contains(out, "v1/index.m3u8?expires=123&signature=abc") {
		t.Errorf("expected URI rewritten with query, got:\n%s", out)
	}
	if !strings.Contains(out, "#EXT-X-STREAM-INF") {
		t.Error("EXT-X-STREAM-INF line was dropped")
	}
}

func TestRewriteManifestURIs_VariantPlaylistSegments(t *testing.T) {
	input := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXTINF:5.992000,
segment000.ts
#EXTINF:5.992000,
segment001.ts
#EXT-X-ENDLIST
`
	out := string(rewriteManifestURIs([]byte(input), "expires=999&signature=xyz"))
	for _, want := range []string{
		"segment000.ts?expires=999&signature=xyz",
		"segment001.ts?expires=999&signature=xyz",
		"#EXT-X-ENDLIST",
		"#EXTINF:5.992000,",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestRewriteManifestURIs_PreservesExistingQuery(t *testing.T) {
	input := "segment000.ts?lang=en\n"
	out := string(rewriteManifestURIs([]byte(input), "expires=1&signature=2"))
	if !strings.Contains(out, "segment000.ts?lang=en&expires=1&signature=2") {
		t.Errorf("expected & separator for existing query, got:\n%s", out)
	}
}

func TestRewriteManifestURIs_LeavesCommentsAndBlanks(t *testing.T) {
	input := "#comment\n\nfoo.ts\n#another\n"
	out := string(rewriteManifestURIs([]byte(input), "q=1"))
	if !strings.Contains(out, "#comment\n") || !strings.Contains(out, "#another") {
		t.Errorf("expected comments preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "foo.ts?q=1") {
		t.Errorf("expected URI rewritten, got:\n%s", out)
	}
	if strings.Contains(out, "#comment?q=1") || strings.Contains(out, "#another?q=1") {
		t.Errorf("comments should not be rewritten:\n%s", out)
	}
}
