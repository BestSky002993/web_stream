// Package urlutil contains tiny URL helpers shared between modules.
// Lives outside both `videos` and `streaming` so neither has to import
// the other for one helper.
package urlutil

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ResolveBaseURL returns the absolute scheme+host to prefix onto API paths.
// Prefers `configured` (typically the PUBLIC_BASE_URL env) when non-empty;
// otherwise derives from the incoming request, honouring X-Forwarded-Proto.
func ResolveBaseURL(c *gin.Context, configured string) string {
	if configured != "" {
		return strings.TrimRight(configured, "/")
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if p := c.GetHeader("X-Forwarded-Proto"); p != "" {
		scheme = p
	}
	return scheme + "://" + c.Request.Host
}

// ThumbnailURL builds the public thumbnail URL for a video ID using the
// canonical `/api/v1/thumbnails/:id` route.
func ThumbnailURL(c *gin.Context, configured, videoID string) string {
	return ResolveBaseURL(c, configured) + "/api/v1/thumbnails/" + videoID
}
