package streaming

import (
	"context"
	"errors"
	"testing"
)

type fakeDomainLister struct {
	domains []string
}

func (f fakeDomainLister) ActiveDomains(context.Context, string) ([]string, error) {
	return f.domains, nil
}

func TestDomainPolicy_EnvAllowsExactHost(t *testing.T) {
	p := NewDomainPolicy("example.com,localhost", nil)
	if err := p.Check(context.Background(), "https://example.com", ""); err != nil {
		t.Fatalf("expected exact env domain pass: %v", err)
	}
	if err := p.Check(context.Background(), "http://localhost:3000", ""); err != nil {
		t.Fatalf("expected localhost origin pass: %v", err)
	}
}

func TestDomainPolicy_ExactHostOnly(t *testing.T) {
	p := NewDomainPolicy("example.com", nil)
	for _, origin := range []string{"https://cdn.example.com", "https://evil-example.com"} {
		if err := p.Check(context.Background(), origin, ""); !errors.Is(err, ErrForbidden) {
			t.Fatalf("expected %s forbidden, got %v", origin, err)
		}
	}
}

func TestDomainPolicy_DatabaseActiveDomainAccepted(t *testing.T) {
	p := NewDomainPolicy("example.com", fakeDomainLister{domains: []string{"testplugin.local"}})
	if err := p.Check(context.Background(), "http://testplugin.local/7-2/", ""); err != nil {
		t.Fatalf("expected db domain pass: %v", err)
	}
}

func TestDomainPolicy_DisabledOrMissingDomainRejected(t *testing.T) {
	p := NewDomainPolicy("example.com", fakeDomainLister{domains: nil})
	if err := p.Check(context.Background(), "http://testplugin.local/7-2/", ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden without active db domain, got %v", err)
	}
}

func TestDomainPolicy_FallsBackToReferer(t *testing.T) {
	p := NewDomainPolicy("", fakeDomainLister{domains: []string{"testplugin.local"}})
	if err := p.Check(context.Background(), "", "http://testplugin.local/7-2/"); err != nil {
		t.Fatalf("expected pass via Referer, got %v", err)
	}
	if err := p.Check(context.Background(), "", "https://evil.com/page"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden via Referer, got %v", err)
	}
}

func TestDomainPolicy_NoHeadersIsAllowed(t *testing.T) {
	p := NewDomainPolicy("example.com", nil)
	if err := p.Check(context.Background(), "", ""); err != nil {
		t.Fatalf("expected nil when no headers present (signature path), got %v", err)
	}
}

func TestDomainPolicy_OriginTakesPrecedence(t *testing.T) {
	p := NewDomainPolicy("example.com", nil)
	err := p.Check(context.Background(), "https://evil.com", "https://example.com/page")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden when Origin fails, got %v", err)
	}
}
