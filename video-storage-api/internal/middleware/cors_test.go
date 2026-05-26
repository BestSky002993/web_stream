package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/alloweddomains"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouterWithCORS(t *testing.T, cfg CORSConfig) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.POST("/echo", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected origin echoed, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials=true, got %q", got)
	}
	methods := w.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !contains(methods, m) {
			t.Errorf("Allow-Methods missing %s: %q", m, methods)
		}
	}
	headers := w.Header().Get("Access-Control-Allow-Headers")
	for _, h := range []string{"Authorization", "Content-Type", "X-API-Key"} {
		if !contains(headers, h) {
			t.Errorf("Allow-Headers missing %s: %q", h, headers)
		}
	}
	if w.Header().Get("Vary") == "" {
		t.Errorf("expected Vary header set")
	}
}

func TestCORS_PreflightDisallowedOrigin(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin for disallowed source, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected no Allow-Credentials, got %q", got)
	}
}

func TestCORS_PreflightNoOriginHeader(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
	})
	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin without Origin request header, got %q", got)
	}
}

func TestCORS_RealRequestAllowedOrigin(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to run (200), got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected origin echoed on real request, got %q", got)
	}
}

func TestCORS_AllowedByDatabaseDomain(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins:   nil,
		AllowCredentials: true,
		DomainPolicy:     alloweddomains.NewPolicy("", corsFakeLister{domains: []string{"testplugin.local"}}, alloweddomains.TypeCORS),
	})

	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	req.Header.Set("Origin", "http://testplugin.local")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://testplugin.local" {
		t.Fatalf("expected db-managed origin echoed, got %q", got)
	}
}

func TestCORS_RealRequestNoOrigin(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin for same-origin/curl request, got %q", got)
	}
}

func TestCORS_CredentialsOff(t *testing.T) {
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: false,
	})

	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected no credentials header when disabled, got %q", got)
	}
}

func TestCORS_NeverWildcardOrigin(t *testing.T) {
	// Even though the user might list multiple origins, the response
	// must echo the specific request origin — never `*`. This keeps
	// the API safe to use with Authorization headers.
	r := newRouterWithCORS(t, CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("Allow-Origin must never be *")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:3000" {
		t.Fatalf("expected exact origin echoed, got %q", got)
	}
}

// contains tells us whether a comma-separated header value includes
// `needle` as one of its entries (case-insensitive, trimmed).
func contains(headerValue, needle string) bool {
	for _, p := range splitAndTrim(headerValue, ',') {
		if equalFold(p, needle) {
			return true
		}
	}
	return false
}

func splitAndTrim(s string, sep byte) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			out = append(out, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, trimSpace(s[start:]))
	return out
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if x >= 'A' && x <= 'Z' {
			x += 'a' - 'A'
		}
		if y >= 'A' && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

type corsFakeLister struct {
	domains []string
}

func (f corsFakeLister) ActiveDomains(context.Context, string) ([]string, error) {
	return f.domains, nil
}
