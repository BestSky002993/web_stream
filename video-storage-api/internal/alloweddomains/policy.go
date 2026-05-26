package alloweddomains

import (
	"context"
	"strings"
)

type ActiveDomainLister interface {
	ActiveDomains(ctx context.Context, kind string) ([]string, error)
}

type Policy struct {
	static []string
	lister ActiveDomainLister
	kind   string
}

func NewPolicy(staticCSV string, lister ActiveDomainLister, kind string) *Policy {
	static := []string{}
	for _, p := range strings.Split(staticCSV, ",") {
		p = strings.TrimSpace(p)
		if p == "" || p == "*" {
			continue
		}
		if strings.Contains(p, "://") {
			if h, ok := HostFromHeader(p); ok {
				p = h
			}
		}
		if d, err := NormalizeDomain(p); err == nil {
			static = append(static, d)
		}
	}
	return &Policy{static: static, lister: lister, kind: kind}
}

func (p *Policy) AllowsHost(ctx context.Context, host string) bool {
	host, err := NormalizeDomain(host)
	if err != nil {
		return false
	}
	for _, d := range p.static {
		if HostMatches(host, d) {
			return true
		}
	}
	if p.lister == nil {
		return false
	}
	active, err := p.lister.ActiveDomains(ctx, p.kind)
	if err != nil {
		return false
	}
	for _, d := range active {
		if HostMatches(host, d) {
			return true
		}
	}
	return false
}

func (p *Policy) AllowsHeader(ctx context.Context, headerValue string) bool {
	host, ok := HostFromHeader(headerValue)
	if !ok {
		return false
	}
	return p.AllowsHost(ctx, host)
}

func (p *Policy) StaticDomains() []string {
	out := make([]string, len(p.static))
	copy(out, p.static)
	return out
}
