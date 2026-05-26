package videos

import (
	"strings"
	"unicode"
)

// NormalizeSlug converts user input into a URL-safe kebab-case slug.
// Same rules as categories.NormalizeSlug — duplicated here to keep the
// videos package self-contained (no inter-feature dependency just for a
// 25-line helper).
func NormalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	b.Grow(len(s))

	prevDash := true
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == '-', r == '_', r == '/', unicode.IsSpace(r):
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			// drop anything else
		}
	}
	return strings.TrimRight(b.String(), "-")
}
