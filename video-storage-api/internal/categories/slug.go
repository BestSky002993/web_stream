package categories

import (
	"strings"
	"unicode"
)

// NormalizeSlug converts user input into a URL-safe kebab-case slug.
//
// Rules:
//   - lower-case
//   - allowed characters: a-z, 0-9
//   - whitespace, '-', '_' and '/' become a single '-'
//   - any other character is stripped
//   - consecutive separators collapse into one '-'
//   - leading and trailing '-' are removed
//
// Returns the empty string if no usable characters remain — callers
// should treat that as a validation error.
func NormalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	b.Grow(len(s))

	// Start in "dashed" state so leading separators are stripped.
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
			// drop everything else (punctuation, non-ASCII letters, etc.)
		}
	}

	return strings.TrimRight(b.String(), "-")
}
