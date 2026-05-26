package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/alloweddomains"
)

// CORSConfig configures the cross-origin policy applied by CORS().
type CORSConfig struct {
	// AllowedOrigins is an exact-match list of Origin header values
	// (scheme + host + optional port). Empty list = enforce CORS but
	// reject all cross-origin requests (no headers emitted, browsers
	// block).
	AllowedOrigins []string

	// AllowCredentials makes the API safe to call with cookies / HTTP
	// auth headers from the listed origins. We always echo the
	// concrete request Origin into Access-Control-Allow-Origin
	// (never `*`), so combining this with the Authorization header is
	// safe under the CORS spec.
	AllowCredentials bool

	// DomainPolicy checks database-managed host allowlists. Static
	// AllowedOrigins stay as bootstrap fallback.
	DomainPolicy *alloweddomains.Policy
}

// CORS returns gin middleware that:
//
//   - echoes a matching Origin into Access-Control-Allow-Origin
//   - answers any OPTIONS request with HTTP 204 before route matching,
//     so preflights never hit a NoRoute 404
//   - sets Vary: Origin on every response (cache-correctness)
//
// Allow-Origin is set to the concrete request Origin (never `*`), so
// this is safe to combine with Authorization or cookie credentials.
//
// Apply this near the front of the global middleware chain, after
// gin.Recovery but before route registration.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[strings.ToLower(o)] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// Always advertise that we vary by Origin so caches don't
		// serve a same-origin response to a cross-origin client.
		c.Header("Vary", "Origin")

		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[strings.ToLower(origin)]; ok || originAllowedByDomainPolicy(c.Request.Context(), cfg.DomainPolicy, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, X-Request-ID, Accept, Range")
				c.Header("Access-Control-Expose-Headers", "X-Request-ID, Content-Range, Content-Length, Accept-Ranges")
				c.Header("Access-Control-Max-Age", "600")
				if cfg.AllowCredentials {
					c.Header("Access-Control-Allow-Credentials", "true")
				}
			}
			// Disallowed origin: emit no Access-Control-* headers.
			// The browser will block the request itself.
		}

		// Preflight short-circuit. Done unconditionally for OPTIONS so
		// it never reaches the NoRoute handler and 404s.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func originAllowedByDomainPolicy(ctx context.Context, policy *alloweddomains.Policy, origin string) bool {
	if policy == nil {
		return false
	}
	return policy.AllowsHeader(ctx, origin)
}
