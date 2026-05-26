package sites

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func domainCtx(headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/api/v1/site/sync/videos", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func TestCheckSiteDomainCorrectHostAllowed(t *testing.T) {
	site := &Site{Domain: "example.com"}
	if err := checkSiteDomain(site, domainCtx(map[string]string{"X-WP-Site-Host": "EXAMPLE.com"})); err != nil {
		t.Fatalf("expected host match: %v", err)
	}
}

func TestCheckSiteDomainMissingProofBlocked(t *testing.T) {
	site := &Site{Domain: "example.com"}
	if err := checkSiteDomain(site, domainCtx(nil)); !errors.Is(err, ErrDomainMismatch) {
		t.Fatalf("expected missing proof blocked, got %v", err)
	}
}

func TestCheckSiteDomainSubdomainMismatchBlocked(t *testing.T) {
	site := &Site{Domain: "example.com"}
	err := checkSiteDomain(site, domainCtx(map[string]string{"Origin": "https://cdn.example.com"}))
	if !errors.Is(err, ErrDomainMismatch) {
		t.Fatalf("expected subdomain mismatch blocked, got %v", err)
	}
}

func TestCheckSiteDomainPathTrickBlocked(t *testing.T) {
	site := &Site{Domain: "example.com"}
	err := checkSiteDomain(site, domainCtx(map[string]string{"X-WP-Site-Host": "example.com/evil.test"}))
	if !errors.Is(err, ErrDomainMismatch) {
		t.Fatalf("expected path trick blocked, got %v", err)
	}
}

func TestCheckSiteDomainSpoofedForwardedHostIgnored(t *testing.T) {
	site := &Site{Domain: "example.com"}
	err := checkSiteDomain(site, domainCtx(map[string]string{
		"Host":             "evil.test",
		"X-Forwarded-Host": "example.com",
	}))
	if !errors.Is(err, ErrDomainMismatch) {
		t.Fatalf("expected x-forwarded-host ignored, got %v", err)
	}
}
