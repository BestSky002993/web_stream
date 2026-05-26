package hls

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/sites"
	"github.com/video-storage-api/internal/streaming"
	"github.com/video-storage-api/internal/urlutil"
)

// HandlerConfig carries the few values the HTTP layer needs at request
// time.
type HandlerConfig struct {
	PublicBaseURL string
}

// Handler serves the public HLS surface plus the JWT- and site-key-
// protected URL issuers.
type Handler struct {
	svc *Service
	cfg HandlerConfig
}

func NewHandler(svc *Service, cfg HandlerConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// ErrorResponse is the canonical JSON envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}

type signedURLResponse struct {
	HLSURL    string    `json:"hls_url"`
	ExpiresAt time.Time `json:"expires_at"`
	Expires   int64     `json:"expires"`
}

// ServeMaster handles GET /api/v1/hls/:video_id/master.m3u8.
// Signature is required and tied to the video_id.
func (h *Handler) ServeMaster(c *gin.Context) {
	videoID := strings.ToLower(c.Param("video_id"))
	if err := ValidateVideoID(videoID); err != nil {
		h.writePathError(c)
		return
	}

	if err := h.verifyRequest(c, videoID); err != nil {
		h.writeSignatureError(c, err)
		return
	}

	cacheKey := "hls:manifest:" + videoID + ":master"
	content, ok := h.svc.CachedBytes(c.Request.Context(), cacheKey)
	if !ok {
		path := h.svc.Paths().MasterManifest(videoID)
		var err error
		content, err = os.ReadFile(path)
		if err != nil {
			h.writeNotFound(c, "master.m3u8 missing on disk")
			return
		}
		h.svc.CacheBytes(c.Request.Context(), cacheKey, content, 5*time.Minute)
	}

	// Rewrite manifest URIs to carry the signature params, so the
	// player's subsequent fetches don't lose authentication.
	query := h.signatureQuery(c)

	out := rewriteManifestURIs(content, query)

	c.Header("Cache-Control", "private, no-cache, max-age=0, must-revalidate")
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", out)
}

// ServeVariantIndex handles GET /api/v1/hls/:video_id/:variant/index.m3u8.
func (h *Handler) ServeVariantIndex(c *gin.Context) {
	videoID := strings.ToLower(c.Param("video_id"))
	variant := c.Param("variant")
	if err := ValidateVideoID(videoID); err != nil {
		h.writePathError(c)
		return
	}
	if err := ValidateVariant(variant); err != nil {
		h.writePathError(c)
		return
	}

	if err := h.verifyRequest(c, videoID); err != nil {
		h.writeSignatureError(c, err)
		return
	}

	cacheKey := "hls:manifest:" + videoID + ":" + variant
	content, ok := h.svc.CachedBytes(c.Request.Context(), cacheKey)
	if !ok {
		path := h.svc.Paths().VariantIndex(videoID, variant)
		var err error
		content, err = os.ReadFile(path)
		if err != nil {
			h.writeNotFound(c, "variant index missing on disk")
			return
		}
		h.svc.CacheBytes(c.Request.Context(), cacheKey, content, 5*time.Minute)
	}

	query := h.signatureQuery(c)

	out := rewriteManifestURIs(content, query)

	if first := firstManifestURI(content); first != "" {
		c.Header("Link", "<"+first+"?"+query+">; rel=preload; as=video")
	}
	c.Header("Cache-Control", "private, no-cache, max-age=0, must-revalidate")
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", out)
}

// ServeSegment handles GET /api/v1/hls/:video_id/:variant/:segment.
// Segments are binary media files served directly with Range support
// via http.ServeContent.
func (h *Handler) ServeSegment(c *gin.Context) {
	videoID := strings.ToLower(c.Param("video_id"))
	variant := c.Param("variant")
	segment := c.Param("segment")
	if err := ValidateVideoID(videoID); err != nil {
		h.writePathError(c)
		return
	}
	if err := ValidateVariant(variant); err != nil {
		h.writePathError(c)
		return
	}
	if err := ValidateSegment(segment); err != nil {
		h.writePathError(c)
		return
	}

	if err := h.verifyRequest(c, videoID); err != nil {
		h.writeSignatureError(c, err)
		return
	}

	path := h.svc.Paths().Segment(videoID, variant, segment)
	f, err := os.Open(path)
	if err != nil {
		h.writeNotFound(c, "segment missing on disk")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		h.writeNotFound(c, "segment stat failed")
		return
	}

	c.Header("Content-Type", segmentContentType(segment))
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Accept-Ranges", "bytes")
	// ServeContent handles Range, If-Modified-Since, ETag, etc.
	http.ServeContent(c.Writer, c.Request, segment, info.ModTime(), f)
}

func segmentContentType(segment string) string {
	switch {
	case strings.HasSuffix(segment, ".ts"):
		return "video/mp2t"
	case strings.HasSuffix(segment, ".m4s"), segment == "init.mp4":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

// IssueAdminURL handles POST /api/v1/admin/videos/:id/hls-url.
func (h *Handler) IssueAdminURL(c *gin.Context) {
	baseURL := urlutil.ResolveBaseURL(c, h.cfg.PublicBaseURL)
	signed, err := h.svc.IssueAdminURL(c.Request.Context(), c.Param("id"), baseURL)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, signedURLResponse{
		HLSURL:    signed.URL,
		ExpiresAt: signed.ExpiresAt,
		Expires:   signed.Expires,
	})
}

// IssueSiteURL handles POST /api/v1/site/videos/:id/hls-url.
func (h *Handler) IssueSiteURL(c *gin.Context) {
	baseURL := urlutil.ResolveBaseURL(c, h.cfg.PublicBaseURL)
	siteID := ""
	if site, ok := sites.SiteFromContext(c); ok {
		siteID = site.ID
		signed, err := h.svc.IssueSiteURL(c.Request.Context(), siteID, site.Domain, c.Param("id"), baseURL)
		if err != nil {
			h.writeServiceError(c, err)
			return
		}
		log.Printf("[SITE_AUDIT] event=signed_url_issued site_id=%s video_id=%s kind=hls ip=%s", siteID, c.Param("id"), c.ClientIP())
		c.JSON(http.StatusOK, signedURLResponse{
			HLSURL:    signed.URL,
			ExpiresAt: signed.ExpiresAt,
			Expires:   signed.Expires,
		})
		return
	}
	signed, err := h.svc.IssueSiteURL(c.Request.Context(), siteID, "", c.Param("id"), baseURL)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, signedURLResponse{
		HLSURL:    signed.URL,
		ExpiresAt: signed.ExpiresAt,
		Expires:   signed.Expires,
	})
}

func (h *Handler) verifyRequest(c *gin.Context, videoID string) error {
	return h.svc.VerifySiteParams(
		videoID,
		c.Query("signature"),
		c.Query("expires"),
		c.Query("site_id"),
		c.Query("domain"),
		c.Query("nonce"),
		c.GetHeader("Origin"),
		c.GetHeader("Referer"),
	)
}

func (h *Handler) signatureQuery(c *gin.Context) string {
	q := url.Values{
		"expires":   {c.Query("expires")},
		"signature": {c.Query("signature")},
	}
	for _, key := range []string{"site_id", "domain", "nonce"} {
		if v := c.Query(key); v != "" {
			q.Set(key, v)
		}
	}
	return q.Encode()
}

// ----- error helpers -----

func (h *Handler) writePathError(c *gin.Context) {
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
}

func (h *Handler) writeSignatureError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, streaming.ErrSignatureRequired):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "signature required"})
	case errors.Is(err, streaming.ErrSignatureExpired):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "signature expired"})
	default:
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "signature invalid"})
	}
}

func (h *Handler) writeNotFound(c *gin.Context, internal string) {
	log.Printf("hls: %s (path=%s)", internal, c.Request.URL.Path)
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
}

func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
	case errors.Is(err, ErrNotReady):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "hls is still processing for this video"})
	default:
		log.Printf("hls: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}

// rewriteManifestURIs walks an HLS manifest line by line and appends
// `query` to every URI line (lines that don't start with `#` and
// aren't empty). Lines already carrying a query string get `&query`
// appended; lines without get `?query`.
func rewriteManifestURIs(content []byte, query string) []byte {
	var out bytes.Buffer
	out.Grow(len(content) + len(query)*8)

	sc := bufio.NewScanner(bytes.NewReader(content))
	// Default Scanner buffer is 64K, fine for normal manifests but bump
	// just in case a long variant list shows up.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
		if strings.Contains(line, "?") {
			out.WriteByte('&')
		} else {
			out.WriteByte('?')
		}
		out.WriteString(query)
		out.WriteByte('\n')
	}
	// Scanner.Err() — log but don't bail (we already wrote what we could).
	if err := sc.Err(); err != nil {
		log.Printf("hls: manifest scan: %v", err)
	}
	_ = io.Discard
	return out.Bytes()
}

func firstManifestURI(content []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}
