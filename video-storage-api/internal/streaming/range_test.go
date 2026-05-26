package streaming

import (
	"errors"
	"testing"
)

func TestParseRange(t *testing.T) {
	const size int64 = 1000

	type tc struct {
		header    string
		wantStart int64
		wantEnd   int64
	}

	good := []tc{
		{"bytes=0-499", 0, 499},
		{"bytes=500-", 500, 999},
		{"bytes=0-", 0, 999},
		{"bytes=999-999", 999, 999},
		{"bytes=-1", 999, 999},
		{"bytes=-1000", 0, 999},
		{"bytes=-10000", 0, 999},  // clamped to size
		{"bytes=0-99999", 0, 999}, // end clamped to size-1
		{"BYTES=0-9", 0, 9},       // case-insensitive prefix
	}
	for _, c := range good {
		r, err := ParseRange(c.header, size)
		if err != nil {
			t.Errorf("ParseRange(%q) unexpected error: %v", c.header, err)
			continue
		}
		if r.Start != c.wantStart || r.End != c.wantEnd {
			t.Errorf("ParseRange(%q) = [%d,%d], want [%d,%d]", c.header, r.Start, r.End, c.wantStart, c.wantEnd)
		}
	}
}

func TestParseRange_Invalid(t *testing.T) {
	const size int64 = 1000

	cases := []string{
		"",
		"abc",
		"items=0-10",
		"bytes=",
		"bytes=0",
		"bytes=-0",
		"bytes=-",
		"bytes=10-5",      // inverted
		"bytes=1000-1001", // start at/past EOF
		"bytes=0-1,2-3",   // multi-range (unsupported)
		"bytes=a-b",
		"bytes=-abc",
	}
	for _, h := range cases {
		if _, err := ParseRange(h, size); !errors.Is(err, ErrInvalidRange) {
			t.Errorf("ParseRange(%q) expected ErrInvalidRange, got %v", h, err)
		}
	}
}

func TestParseRange_ZeroSizeFile(t *testing.T) {
	if _, err := ParseRange("bytes=0-10", 0); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("expected ErrInvalidRange for empty file, got %v", err)
	}
}

func TestByteRange_Length(t *testing.T) {
	r := ByteRange{Start: 0, End: 499}
	if r.Length() != 500 {
		t.Fatalf("expected 500, got %d", r.Length())
	}
}
