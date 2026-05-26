package streaming

import (
	"net/url"
	"strconv"
	"strings"
)

func parseExpires(raw string) (int64, error) {
	if raw == "" {
		return 0, ErrSignatureRequired
	}
	expires, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, ErrSignatureInvalid
	}
	return expires, nil
}

func ParseExpiresForHLS(raw string) (int64, error) {
	return parseExpires(raw)
}

func checkSignedRequestDomain(allowed, origin, referer string) error {
	allowedHost, ok := signedHost(allowed)
	if !ok {
		return ErrSignatureInvalid
	}
	for _, raw := range []string{origin, referer} {
		if raw == "" {
			continue
		}
		h, ok := signedHost(raw)
		if !ok || h != allowedHost {
			return ErrForbidden
		}
		return nil
	}
	return nil
}

func CheckSignedRequestDomainForHLS(allowed, origin, referer string) error {
	return checkSignedRequestDomain(allowed, origin, referer)
}

func signedHost(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return "", false
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			return "", false
		}
		raw = u.Hostname()
	} else {
		if strings.ContainsAny(raw, "/?#@") {
			return "", false
		}
		u, err := url.Parse("//" + raw)
		if err != nil || u.Hostname() == "" {
			return "", false
		}
		raw = u.Hostname()
	}
	raw = strings.TrimSuffix(strings.ToLower(raw), ".")
	if raw == "" || strings.Contains(raw, "..") || strings.HasPrefix(raw, ".") {
		return "", false
	}
	return raw, true
}
