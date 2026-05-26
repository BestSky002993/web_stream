package streaming

import (
	"strconv"
	"strings"
)

// ByteRange is a half-open interval clamped to the file's byte range.
type ByteRange struct {
	Start int64
	End   int64 // inclusive
}

// Length returns the number of bytes in the range.
func (r ByteRange) Length() int64 { return r.End - r.Start + 1 }

// ParseRange parses a single-range "bytes=…" header (RFC 7233 §3.1).
//
// Supported forms:
//
//	bytes=0-499      → first 500 bytes
//	bytes=500-       → from byte 500 to EOF
//	bytes=-500       → last 500 bytes
//
// Multipart ranges (comma-separated) are intentionally NOT supported in
// this phase — we return ErrInvalidRange for them so the caller can
// emit 416.
//
// `size` is the total file size. If the requested range is entirely
// outside the file, ParseRange returns ErrInvalidRange.
func ParseRange(header string, size int64) (ByteRange, error) {
	if size <= 0 {
		return ByteRange{}, ErrInvalidRange
	}
	if !strings.HasPrefix(strings.ToLower(header), "bytes=") {
		return ByteRange{}, ErrInvalidRange
	}
	spec := strings.TrimSpace(header[len("bytes="):])
	if spec == "" || strings.Contains(spec, ",") {
		return ByteRange{}, ErrInvalidRange
	}

	dash := strings.IndexByte(spec, '-')
	if dash < 0 {
		return ByteRange{}, ErrInvalidRange
	}
	startStr := strings.TrimSpace(spec[:dash])
	endStr := strings.TrimSpace(spec[dash+1:])

	var start, end int64

	switch {
	case startStr == "":
		// Suffix form: bytes=-N
		if endStr == "" {
			return ByteRange{}, ErrInvalidRange
		}
		n, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || n <= 0 {
			return ByteRange{}, ErrInvalidRange
		}
		if n > size {
			n = size
		}
		start = size - n
		end = size - 1

	default:
		s, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil || s < 0 {
			return ByteRange{}, ErrInvalidRange
		}
		start = s

		if endStr == "" {
			end = size - 1
		} else {
			e, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil || e < 0 {
				return ByteRange{}, ErrInvalidRange
			}
			end = e
			if end >= size {
				end = size - 1
			}
		}
	}

	if start > end || start >= size {
		return ByteRange{}, ErrInvalidRange
	}
	return ByteRange{Start: start, End: end}, nil
}
