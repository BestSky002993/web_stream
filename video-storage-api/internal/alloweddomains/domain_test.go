package alloweddomains

import (
	"context"
	"testing"
)

type fakeLister struct {
	domains []string
}

func (f fakeLister) ActiveDomains(context.Context, string) ([]string, error) {
	return f.domains, nil
}

func TestNormalizeDomain(t *testing.T) {
	tests := map[string]string{
		" Example.COM ":      "example.com",
		"localhost":          "localhost",
		"testplugin.local/":  "",
		"http://example.com": "",
		"example.com/path":   "",
		"*":                  "",
		"":                   "",
	}
	for in, want := range tests {
		got, err := NormalizeDomain(in)
		if want == "" {
			if err == nil {
				t.Fatalf("NormalizeDomain(%q) expected error, got %q", in, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Fatalf("NormalizeDomain(%q)=%q,%v want %q,nil", in, got, err, want)
		}
	}
}

func TestHostExtraction(t *testing.T) {
	tests := map[string]string{
		"https://example.com/path":       "example.com",
		"http://testplugin.local/7-2/":   "testplugin.local",
		"http://localhost:8088/admin":    "localhost",
		"testplugin.local":               "testplugin.local",
		"https://example.com/path?q=yes": "example.com",
	}
	for in, want := range tests {
		got, ok := HostFromHeader(in)
		if !ok || got != want {
			t.Fatalf("HostFromHeader(%q)=%q,%v want %q,true", in, got, ok, want)
		}
	}
}

func TestHostMatchesExactOnly(t *testing.T) {
	if !HostMatches("example.com", "example.com") {
		t.Fatal("expected exact match")
	}
	if HostMatches("evil-example.com", "example.com") {
		t.Fatal("evil-example.com must not match example.com")
	}
	if HostMatches("cdn.example.com", "example.com") {
		t.Fatal("subdomains must not match without explicit support")
	}
}

func TestPolicyMergesEnvAndDatabase(t *testing.T) {
	p := NewPolicy("example.com", fakeLister{domains: []string{"testplugin.local"}}, TypeStream)
	if !p.AllowsHost(context.Background(), "example.com") {
		t.Fatal("expected env domain accepted")
	}
	if !p.AllowsHost(context.Background(), "testplugin.local") {
		t.Fatal("expected db domain accepted")
	}
	if p.AllowsHost(context.Background(), "evil.com") {
		t.Fatal("expected unknown domain rejected")
	}
}
