package streaming

import (
	"context"

	"github.com/video-storage-api/internal/alloweddomains"
)

// DomainPolicy enforces hotlink protection by checking Origin/Referer
// hosts against the static env bootstrap list plus database-managed
// active domains with type stream/both.
//
// Matching is exact-host only. An allowed example.com does not match
// cdn.example.com or evil-example.com.
type DomainPolicy struct {
	policy *alloweddomains.Policy
}

func NewDomainPolicy(raw string, lister alloweddomains.ActiveDomainLister) *DomainPolicy {
	return &DomainPolicy{policy: alloweddomains.NewPolicy(raw, lister, alloweddomains.TypeStream)}
}

func (p *DomainPolicy) AllowAll() bool { return false }

func (p *DomainPolicy) Allowed() []string {
	if p == nil || p.policy == nil {
		return nil
	}
	return p.policy.StaticDomains()
}

// Check returns nil if the request is allowed, ErrForbidden otherwise.
// If neither Origin nor Referer is present, it allows the request so
// direct server-to-server signed URL fetches continue to work.
func (p *DomainPolicy) Check(ctx context.Context, origin, referer string) error {
	if p == nil || p.policy == nil {
		return ErrForbidden
	}
	if h, ok := alloweddomains.HostFromHeader(origin); ok {
		if p.policy.AllowsHost(ctx, h) {
			return nil
		}
		return ErrForbidden
	}
	if h, ok := alloweddomains.HostFromHeader(referer); ok {
		if p.policy.AllowsHost(ctx, h) {
			return nil
		}
		return ErrForbidden
	}
	return nil
}
