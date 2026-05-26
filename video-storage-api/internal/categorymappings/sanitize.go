package categorymappings

import (
	"net/url"
	"strings"
	"unicode"
)

func NormalizeRemoteCategory(in RemoteCategoryImport) (RemoteCategoryImport, error) {
	remoteTermID := cleanText(in.RemoteTermID)
	remoteName := cleanText(in.RemoteName)
	remoteTaxonomy := normalizeSlugLike(in.RemoteTaxonomy)
	remoteSlug, err := normalizeRemoteSlug(in.RemoteSlug)
	if err != nil {
		return RemoteCategoryImport{}, err
	}

	out := RemoteCategoryImport{
		RemoteTermID:       remoteTermID,
		RemoteSlug:         remoteSlug,
		RemoteName:         remoteName,
		RemoteTaxonomy:     remoteTaxonomy,
		ParentRemoteTermID: cleanOptional(in.ParentRemoteTermID),
	}
	if out.RemoteTermID == "" {
		return RemoteCategoryImport{}, invalidCategory("remote_term_id", "empty after sanitization")
	}
	if out.RemoteName == "" {
		return RemoteCategoryImport{}, invalidCategory("remote_name", "empty after sanitization")
	}
	if out.RemoteTaxonomy == "" {
		return RemoteCategoryImport{}, invalidCategory("remote_taxonomy", "empty after sanitization")
	}
	if !validSlugLike(out.RemoteTaxonomy) {
		return RemoteCategoryImport{}, invalidCategory("remote_taxonomy", "unsafe taxonomy slug")
	}
	return out, nil
}

func normalizeRemoteSlug(raw string) (string, error) {
	slug := cleanText(raw)
	if decoded, err := url.PathUnescape(slug); err == nil {
		slug = decoded
	} else if decoded, err := url.QueryUnescape(slug); err == nil {
		slug = decoded
	}
	slug = normalizeSlugLike(slug)
	if slug == "" {
		return "", invalidCategory("remote_slug", "empty after sanitization")
	}
	if !validSlugLike(slug) {
		return "", invalidCategory("remote_slug", "unsafe slug")
	}
	return slug, nil
}

func normalizeSlugLike(s string) string {
	s = strings.ToLower(cleanText(s))
	s = strings.Join(strings.Fields(s), "-")
	return s
}

func cleanOptional(s *string) *string {
	if s == nil {
		return nil
	}
	v := cleanText(*s)
	if v == "" {
		return nil
	}
	return &v
}

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

func validSlugLike(s string) bool {
	if s == "" || len([]rune(s)) > 200 {
		return false
	}
	if strings.Contains(s, "://") || strings.HasPrefix(s, "//") {
		return false
	}
	for _, r := range s {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsMark(r):
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func sameNameKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(cleanText(s)), " "))
}
