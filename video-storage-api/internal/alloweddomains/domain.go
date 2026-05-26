package alloweddomains

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	TypeStream = "stream"
	TypeCORS   = "cors"
	TypeBoth   = "both"

	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var (
	ErrNotFound      = errors.New("allowed domain not found")
	ErrDomainTaken   = errors.New("domain already exists")
	ErrInvalidDomain = errors.New("invalid domain")
	ErrInvalidType   = errors.New("invalid domain type")
	ErrInvalidStatus = errors.New("invalid domain status")
)

type Domain struct {
	ID        string
	Domain    string
	Type      string
	Status    string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NormalizeDomain lowercases and trims a dashboard-entered domain. It
// accepts host-only values such as example.com, localhost, and
// testplugin.local. Schemes, wildcards, paths, queries, fragments, and
// whitespace are rejected to avoid URL confusion.
func NormalizeDomain(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	if d == "" || d == "*" {
		return "", ErrInvalidDomain
	}
	if strings.ContainsAny(d, " \t\r\n") {
		return "", ErrInvalidDomain
	}
	if strings.Contains(d, "://") {
		return "", ErrInvalidDomain
	}
	if strings.ContainsAny(d, "/?#") {
		return "", ErrInvalidDomain
	}
	d = strings.TrimSuffix(d, ".")
	if d == "" || d == "*" {
		return "", ErrInvalidDomain
	}
	host := d
	if h, _, err := net.SplitHostPort(d); err == nil {
		host = h
	} else if strings.Contains(d, ":") {
		return "", ErrInvalidDomain
	}
	if host == "" || host == "*" || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return "", ErrInvalidDomain
	}
	return host, nil
}

func IsValidType(t string) bool {
	return t == TypeStream || t == TypeCORS || t == TypeBoth
}

func IsValidStatus(s string) bool {
	return s == StatusActive || s == StatusDisabled
}

func HostFromHeader(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	if u, err := url.Parse(v); err == nil && u.Hostname() != "" {
		return strings.ToLower(u.Hostname()), true
	}
	host, err := NormalizeDomain(v)
	if err != nil {
		return "", false
	}
	return host, true
}

// HostMatches is intentionally exact. Subdomain matching can be added
// later as an explicit per-domain setting; for now example.com does not
// match cdn.example.com or evil-example.com.
func HostMatches(host, allowed string) bool {
	host, err := NormalizeDomain(host)
	if err != nil {
		return false
	}
	allowed, err = NormalizeDomain(allowed)
	if err != nil {
		return false
	}
	return host == allowed
}
