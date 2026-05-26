package sites

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// HeaderAPIKey is the HTTP header carrying the site API key.
const HeaderAPIKey = "X-API-Key"

// siteContextKey is the gin.Context key used to stash the authenticated
// site. Unexported so callers must use SiteFromContext.
const siteContextKey = "sites.site"

// RequireSite returns the gin middleware that protects site endpoints.
// It:
//
//   - reads X-API-Key
//   - validates against the stored hash (and status)
//   - validates Origin / Referer against the site's domain
//   - stashes the site into the request context
//   - emits an access-log line on the way out
func RequireSite(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(HeaderAPIKey)

		site, err := svc.Authenticate(c.Request.Context(), key)
		if err != nil {
			writeMiddlewareError(c, err)
			return
		}

		if derr := checkSiteDomain(site, c); derr != nil {
			writeMiddlewareError(c, derr)
			return
		}

		c.Set(siteContextKey, site)
		defer logSiteRequest(c, site)
		c.Next()
	}
}

// SiteFromContext returns the site stashed by RequireSite.
func SiteFromContext(c *gin.Context) (*Site, bool) {
	v, ok := c.Get(siteContextKey)
	if !ok {
		return nil, false
	}
	s, ok := v.(*Site)
	return s, ok
}

// checkSiteDomain enforces the per-site domain whitelist:
//   - WordPress server-side calls must send X-WP-Site-Host.
//   - Browser/player calls are checked with Origin/Referer first.
//   - Host is only a fallback for same-origin server calls.
//   - X-Forwarded-Host is not trusted here; nginx/proxy config owns that.
//   - matching is exact-host only after lowercasing and stripping scheme/path/port.
func checkSiteDomain(site *Site, c *gin.Context) error {
	allowed, ok := normalizeHost(site.Domain)
	if !ok {
		return ErrDomainMismatch
	}
	candidates := []string{
		c.GetHeader("X-WP-Site-Host"),
		c.GetHeader("Origin"),
		c.GetHeader("Referer"),
		c.GetHeader("Host"),
	}
	for _, raw := range candidates {
		if raw == "" {
			continue
		}
		host, ok := normalizeHost(raw)
		if !ok {
			log.Printf("[SITE_AUDIT] event=domain_mismatch_blocked site_id=%s registered_domain=%s supplied_domain=%q path=%s ip=%s", site.ID, allowed, raw, c.Request.URL.Path, c.ClientIP())
			return ErrDomainMismatch
		}
		if host == allowed {
			return nil
		}
		log.Printf("[SITE_AUDIT] event=domain_mismatch_blocked site_id=%s registered_domain=%s supplied_domain=%s path=%s ip=%s", site.ID, allowed, host, c.Request.URL.Path, c.ClientIP())
		return ErrDomainMismatch
	}
	log.Printf("[SITE_AUDIT] event=domain_mismatch_blocked site_id=%s registered_domain=%s supplied_domain=missing path=%s ip=%s", site.ID, allowed, c.Request.URL.Path, c.ClientIP())
	return ErrDomainMismatch
}

func normalizeHost(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	if strings.ContainsAny(v, " \t\r\n") {
		return "", false
	}
	if strings.Contains(v, "://") {
		u, err := url.Parse(v)
		if err != nil || u.Hostname() == "" {
			return "", false
		}
		v = u.Hostname()
	} else {
		if strings.ContainsAny(v, "/?#@") {
			return "", false
		}
		if u, err := url.Parse("//" + v); err == nil && u.Hostname() != "" {
			v = u.Hostname()
		}
	}
	v = strings.TrimSuffix(strings.ToLower(v), ".")
	if v == "" || strings.Contains(v, "..") || strings.HasPrefix(v, ".") {
		return "", false
	}
	return v, true
}

func writeMiddlewareError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrMissingAPIKey):
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "api key required"})
	case errors.Is(err, ErrInvalidAPIKey):
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid api key"})
	case errors.Is(err, ErrSiteDisabled):
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "site is disabled"})
	case errors.Is(err, ErrSiteSuspended):
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "site is suspended"})
	case errors.Is(err, ErrDomainMismatch):
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "domain not authorized for this site key"})
	default:
		log.Printf("sites: middleware error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}

// logSiteRequest emits one line per site-protected request. Format
// mirrors the [STREAM] log so they're easy to grep together.
func logSiteRequest(c *gin.Context, site *Site) {
	log.Printf("[SITE] site_id=%s domain=%s method=%s path=%s status=%d ip=%s ua=%q",
		site.ID,
		site.Domain,
		c.Request.Method,
		c.Request.URL.Path,
		c.Writer.Status(),
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
}
