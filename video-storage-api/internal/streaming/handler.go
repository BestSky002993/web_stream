package streaming

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/categorymappings"
	"github.com/video-storage-api/internal/publications"
	"github.com/video-storage-api/internal/sites"
	"github.com/video-storage-api/internal/urlutil"
	"github.com/video-storage-api/internal/videos"
)

// Handler is the HTTP layer for the streaming endpoints.
type Handler struct {
	svc *Service
	cfg HandlerConfig
}

// HandlerConfig captures the bits of streaming config the handler needs
// at request time (publicBaseURL for signed-URL responses).
type HandlerConfig struct {
	PublicBaseURL string
}

func NewHandler(svc *Service, cfg HandlerConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// ErrorResponse is the canonical error envelope for streaming endpoints.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Stream handles GET/HEAD /api/v1/stream/:videoID.
//
// Flow:
//  1. authorise (domain whitelist → DB lookup → status → visibility/signature)
//  2. open file from private storage
//  3. parse Range header (if any) and serve 200 full or 206 partial
//  4. emit a structured access log line on completion
func (h *Handler) Stream(c *gin.Context) {
	videoID := c.Param("id")
	start := time.Now()
	var statusCode int
	var bytesWritten int64

	// access log on the way out
	defer func() {
		log.Printf("[STREAM] video_id=%s ip=%s ua=%q status=%d bytes=%d range=%q took=%s",
			videoID,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			statusCode,
			bytesWritten,
			c.GetHeader("Range"),
			time.Since(start),
		)
	}()

	v, err := h.svc.AuthorizeStream(c.Request.Context(), AccessParams{
		VideoID:   videoID,
		Origin:    c.GetHeader("Origin"),
		Referer:   c.GetHeader("Referer"),
		Signature: c.Query("signature"),
		Expires:   c.Query("expires"),
		SiteID:    c.Query("site_id"),
		Domain:    c.Query("domain"),
		Nonce:     c.Query("nonce"),
	})
	if err != nil {
		statusCode = writeAuthzError(c, err)
		return
	}

	file, size, err := h.svc.OpenForStream(c.Request.Context(), v)
	if err != nil {
		if errors.Is(err, ErrFileMissing) {
			log.Printf("streaming: file missing for video_id=%s filename=%s", v.ID, v.VideoFilename)
		}
		statusCode = writeServiceError(c, err)
		return
	}
	defer file.Close()

	contentType := chooseContentType(v)

	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", contentType)
	// Cache directive: the file content is immutable per filename
	// (random hex), so caches can hold the response for a while.
	c.Header("Cache-Control", "private, max-age=300")

	rangeHeader := c.GetHeader("Range")
	isHead := c.Request.Method == http.MethodHead

	if rangeHeader == "" {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
		c.Status(http.StatusOK)
		statusCode = http.StatusOK
		if isHead {
			return
		}
		n, _ := io.Copy(c.Writer, file)
		bytesWritten = n
		return
	}

	r, err := ParseRange(rangeHeader, size)
	if err != nil {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
		c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
		statusCode = http.StatusRequestedRangeNotSatisfiable
		return
	}

	if _, err := file.Seek(r.Start, io.SeekStart); err != nil {
		statusCode = writeServiceError(c, fmt.Errorf("seek: %w", err))
		return
	}

	length := r.Length()
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", r.Start, r.End, size))
	c.Header("Content-Length", strconv.FormatInt(length, 10))
	c.Status(http.StatusPartialContent)
	statusCode = http.StatusPartialContent

	if isHead {
		return
	}
	n, _ := io.CopyN(c.Writer, file, length)
	bytesWritten = n
}

// PublicMetadata handles GET /api/v1/videos/:slug.
func (h *Handler) PublicMetadata(c *gin.Context) {
	v, err := h.svc.LookupPublicMetadata(c.Request.Context(), PublicMetadataParams{
		Slug:    c.Param("slug"),
		Origin:  c.GetHeader("Origin"),
		Referer: c.GetHeader("Referer"),
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}

	c.JSON(http.StatusOK, h.toPublicResponse(c, v))
}

// PublicList handles GET /api/v1/videos.
func (h *Handler) PublicList(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.svc.ListPublic(c.Request.Context(), PublicListParams{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		CategoryID: c.Query("category"),
		Sort:       c.Query("sort"),
		Origin:     c.GetHeader("Origin"),
		Referer:    c.GetHeader("Referer"),
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}

	c.JSON(http.StatusOK, PublicListResponse{
		Data: h.toPublicResponseList(c, result.Items),
		Pagination: PublicPagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// PublicRandom handles GET /api/v1/videos/random.
func (h *Handler) PublicRandom(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.svc.RandomPublic(c.Request.Context(), RandomPublicParams{
		Limit:   limit,
		Seed:    c.Query("seed"),
		Origin:  c.GetHeader("Origin"),
		Referer: c.GetHeader("Referer"),
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}

	c.JSON(http.StatusOK, RandomListResponse{
		Data: h.toPublicResponseList(c, result.Items),
		Seed: result.SeedKey,
	})
}

// Thumbnail handles GET/HEAD /api/v1/thumbnails/:videoID.
func (h *Handler) Thumbnail(c *gin.Context) {
	v, f, size, err := h.svc.OpenThumbnailForVideo(c.Request.Context(), ThumbnailParams{
		VideoID: c.Param("id"),
		Origin:  c.GetHeader("Origin"),
		Referer: c.GetHeader("Referer"),
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}
	defer f.Close()

	c.Header("Content-Type", contentTypeForThumbnail(extOf(v.ThumbnailFilename)))
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	// Thumbnails change only when the admin regenerates them, which
	// produces a new random filename — so we can cache for a while.
	c.Header("Cache-Control", "public, max-age=300")

	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusOK)
	io.Copy(c.Writer, f)
}

// IssueSignedURL handles POST /api/v1/admin/videos/:id/stream-url.
// Already protected by the admin JWT middleware (see module wiring).
func (h *Handler) IssueSignedURL(c *gin.Context) {
	baseURL := h.resolveBaseURL(c)

	signed, err := h.svc.IssueSignedURL(c.Request.Context(), c.Param("id"), baseURL)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, signedURLResponse{
		StreamURL: signed.URL,
		ExpiresAt: signed.ExpiresAt,
		Expires:   signed.Expires,
	})
}

// SiteList handles GET /api/v1/site/videos.
// Same shape as PublicList; the X-API-Key middleware has already
// validated the site and stashed it in context.
func (h *Handler) SiteList(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.svc.ListPublic(c.Request.Context(), PublicListParams{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		CategoryID: c.Query("category"),
		Sort:       c.Query("sort"),
		SiteID:     siteIDFromContext(c),
		// Origin/Referer already enforced by the site middleware per-site;
		// the global ALLOWED_STREAM_DOMAINS check should not also fire here.
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}

	c.JSON(http.StatusOK, PublicListResponse{
		Data: h.toPublicResponseList(c, result.Items),
		Pagination: PublicPagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// SiteRandom handles GET /api/v1/site/videos/random.
// If the client doesn't supply a `seed`, the site's ID is used so each
// WordPress install gets its own stable daily order automatically.
func (h *Handler) SiteRandom(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))

	seed := c.Query("seed")
	if seed == "" {
		if site, ok := sites.SiteFromContext(c); ok {
			seed = site.ID
		}
	}

	result, err := h.svc.RandomPublic(c.Request.Context(), RandomPublicParams{
		Limit:  limit,
		Seed:   seed,
		SiteID: siteIDFromContext(c),
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}

	c.JSON(http.StatusOK, RandomListResponse{
		Data: h.toPublicResponseList(c, result.Items),
		Seed: result.SeedKey,
	})
}

// SiteSyncVideos handles GET /api/v1/site/sync/videos.
func (h *Handler) SiteSyncVideos(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	siteID := siteIDFromContext(c)
	pubs, total, err := h.svc.ListSitePublications(c.Request.Context(), siteID, PublicListParams{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		CategoryID: c.Query("category"),
		Sort:       c.Query("sort"),
	})
	if err != nil {
		writeAuthzError(c, err)
		return
	}
	if page < 1 {
		page = 1
	}
	limit = ClampLimit(limit, 20, 100)
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	mappingRows, err := h.svc.MappingStatusForPublications(c.Request.Context(), siteID, pubs)
	if err != nil {
		writeAuthzError(c, err)
		return
	}
	c.JSON(http.StatusOK, SyncVideoListResponse{
		Data: h.toSyncResponseList(c, pubs, mappingRows),
		Pagination: PublicPagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// SiteSyncCategories handles GET /api/v1/site/sync/categories.
func (h *Handler) SiteSyncCategories(c *gin.Context) {
	cats, err := h.svc.ListSyncCategories(c.Request.Context())
	if err != nil {
		writeAuthzError(c, err)
		return
	}
	out := make([]PublicCategoryBrief, len(cats))
	for i := range cats {
		out[i] = PublicCategoryBrief{ID: cats[i].ID, Name: cats[i].Name}
	}
	c.JSON(http.StatusOK, SyncCategoryListResponse{Data: out})
}

// SiteSyncStatus lets the WordPress plugin record the post it created
// without exposing admin credentials.
func (h *Handler) SiteSyncStatus(c *gin.Context) {
	var req SyncStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	p, err := h.svc.UpdateSyncResult(c.Request.Context(), siteIDFromContext(c), c.Param("id"), req.Status, req.RemotePostID, req.RemoteSlug)
	if err != nil {
		writeAuthzError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"publication_status": p.Status,
		"remote_post_id":     p.RemotePostID,
		"remote_slug":        p.RemoteSlug,
		"last_synced_at":     p.LastSyncedAt,
	})
}

// SiteIssueSignedURL handles POST /api/v1/site/videos/:id/stream-url.
// Restricted to public+ready videos so a site cannot use a known ID to
// stream private content.
func (h *Handler) SiteIssueSignedURL(c *gin.Context) {
	baseURL := h.resolveBaseURL(c)

	site, _ := sites.SiteFromContext(c)
	siteID, siteDomain := "", ""
	if site != nil {
		siteID = site.ID
		siteDomain = site.Domain
	}
	signed, err := h.svc.IssuePublicSignedURL(c.Request.Context(), siteID, siteDomain, c.Param("id"), baseURL)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	log.Printf("[SITE_AUDIT] event=signed_url_issued site_id=%s video_id=%s kind=mp4 ip=%s", siteID, c.Param("id"), c.ClientIP())

	c.JSON(http.StatusOK, signedURLResponse{
		StreamURL: signed.URL,
		ExpiresAt: signed.ExpiresAt,
		Expires:   signed.Expires,
	})
}

// toPublicResponse converts a Video into the public JSON shape.
func (h *Handler) toPublicResponse(c *gin.Context, v *videos.Video) PublicVideoResponse {
	categories, categoryIDs := publicCategories(v)
	return PublicVideoResponse{
		ID:          v.ID,
		Title:       v.Title,
		Slug:        v.Slug,
		Description: v.Description,
		Category: PublicCategoryBrief{
			ID:   v.CategoryID,
			Name: v.CategoryName,
		},
		Categories:      categories,
		CategoryIDs:     categoryIDs,
		ThumbnailURL:    urlutil.ThumbnailURL(c, h.cfg.PublicBaseURL, v.ID),
		DurationSeconds: v.DurationSeconds,
		ViewsCount:      v.ViewsCount,
		CreatedAt:       v.CreatedAt,
		StreamAvailable: v.Status == videos.StatusReady,
	}
}

func (h *Handler) toPublicResponseList(c *gin.Context, vs []videos.Video) []PublicVideoResponse {
	out := make([]PublicVideoResponse, len(vs))
	for i := range vs {
		out[i] = h.toPublicResponse(c, &vs[i])
	}
	return out
}

func (h *Handler) toSyncResponseList(c *gin.Context, pubs []publications.Publication, mappingRows map[string]categorymappings.MappingRow) []SyncVideoResponse {
	out := make([]SyncVideoResponse, 0, len(pubs))
	for i := range pubs {
		p := &pubs[i]
		if p.Video == nil {
			continue
		}
		v := p.Video
		mappingStatus := categorymappings.MappingStatusUnmapped
		syncMode := categorymappings.SyncModeBlock
		var mappedRemote *SyncRemoteCategory
		categories, categoryIDs := publicCategories(v)
		selectedCategoryID := v.CategoryID
		selectedCategoryName := v.CategoryName
		if len(categories) > 0 {
			selectedCategoryID = categories[0].ID
			selectedCategoryName = categories[0].Name
		}
		var selectedRow *categorymappings.MappingRow
		for _, cat := range categories {
			if row, ok := mappingRows[cat.ID]; ok {
				if selectedRow == nil {
					rowCopy := row
					selectedRow = &rowCopy
					selectedCategoryID = cat.ID
					selectedCategoryName = cat.Name
				}
				if row.Status == categorymappings.MappingStatusMapped {
					rowCopy := row
					selectedRow = &rowCopy
					selectedCategoryID = cat.ID
					selectedCategoryName = cat.Name
					break
				}
			}
		}
		if selectedRow != nil {
			row := *selectedRow
			mappingStatus = row.Status
			if row.Mapping != nil {
				syncMode = row.Mapping.SyncMode
			}
			if p.SyncMode != nil {
				syncMode = *p.SyncMode
			}
			if row.Mapping != nil && row.Mapping.Enabled && row.RemoteCategory != nil {
				mappedRemote = &SyncRemoteCategory{
					ID:             row.RemoteCategory.ID,
					RemoteTermID:   row.RemoteCategory.RemoteTermID,
					RemoteSlug:     row.RemoteCategory.RemoteSlug,
					RemoteName:     row.RemoteCategory.RemoteName,
					RemoteTaxonomy: row.RemoteCategory.RemoteTaxonomy,
				}
			}
		}
		out = append(out, SyncVideoResponse{
			VideoID:               v.ID,
			Title:                 v.Title,
			Slug:                  v.Slug,
			Description:           v.Description,
			ThumbnailURL:          urlutil.ThumbnailURL(c, h.cfg.PublicBaseURL, v.ID),
			Category:              PublicCategoryBrief{ID: selectedCategoryID, Name: selectedCategoryName},
			Categories:            categories,
			CategoryIDs:           categoryIDs,
			BackendCategoryID:     selectedCategoryID,
			BackendCategoryName:   selectedCategoryName,
			CategoryMappingStatus: mappingStatus,
			MappedRemoteCategory:  mappedRemote,
			SyncMode:              syncMode,
			CategoryMapping:       p.CategoryMapping,
			Tags:                  p.Tags,
			Actors:                []string{},
			DurationSeconds:       v.DurationSeconds,
			UpdatedAt:             v.UpdatedAt,
			PublicationStatus:     p.Status,
			RemotePostID:          p.RemotePostID,
			RemoteSlug:            p.RemoteSlug,
		})
	}
	return out
}

func publicCategories(v *videos.Video) ([]PublicCategoryBrief, []string) {
	categories := make([]PublicCategoryBrief, 0, len(v.Categories))
	categoryIDs := make([]string, 0, len(v.Categories))
	for _, cat := range v.Categories {
		categories = append(categories, PublicCategoryBrief{ID: cat.ID, Name: cat.Name})
		categoryIDs = append(categoryIDs, cat.ID)
	}
	if len(categories) == 0 && v.CategoryID != "" {
		categories = append(categories, PublicCategoryBrief{ID: v.CategoryID, Name: v.CategoryName})
		categoryIDs = append(categoryIDs, v.CategoryID)
	}
	return categories, categoryIDs
}

func siteIDFromContext(c *gin.Context) string {
	if site, ok := sites.SiteFromContext(c); ok {
		return site.ID
	}
	return ""
}

type signedURLResponse struct {
	StreamURL string    `json:"stream_url"`
	ExpiresAt time.Time `json:"expires_at"`
	Expires   int64     `json:"expires"`
}

// chooseContentType prefers the MIME we sniffed at upload time and
// falls back to the file extension. Unknown values become
// application/octet-stream so the browser won't try to render
// something the wrong way.
func chooseContentType(v *videos.Video) string {
	if v.VideoMimeType != "" {
		return v.VideoMimeType
	}
	return contentTypeForExt(extOf(v.VideoFilename))
}

// resolveBaseURL delegates to the shared urlutil helper so streaming
// and videos compute identical base URLs.
func (h *Handler) resolveBaseURL(c *gin.Context) string {
	return urlutil.ResolveBaseURL(c, h.cfg.PublicBaseURL)
}

// writeAuthzError maps authorisation-side errors to status codes.
// Used by stream / metadata / thumbnail endpoints.
func writeAuthzError(c *gin.Context, err error) int {
	switch {
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "forbidden"})
		return http.StatusForbidden
	case errors.Is(err, ErrSignatureRequired):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "signature required"})
		return http.StatusUnauthorized
	case errors.Is(err, ErrSignatureExpired):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "signature expired"})
		return http.StatusUnauthorized
	case errors.Is(err, ErrSignatureInvalid):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "signature invalid"})
		return http.StatusUnauthorized
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
		return http.StatusNotFound
	case errors.Is(err, publications.ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
		return http.StatusNotFound
	case errors.Is(err, ErrNotAvailable):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not available"})
		return http.StatusNotFound
	case errors.Is(err, ErrFileMissing):
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return http.StatusInternalServerError
	default:
		log.Printf("streaming: authz error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return http.StatusInternalServerError
	}
}

// writeServiceError is used after authorisation has passed but a later
// step fails. Same shape as writeAuthzError but distinct so we can
// later log them differently if needed.
func writeServiceError(c *gin.Context, err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
		return http.StatusNotFound
	case errors.Is(err, ErrNotAvailable):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not available"})
		return http.StatusNotFound
	case errors.Is(err, ErrFileMissing):
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return http.StatusInternalServerError
	default:
		log.Printf("streaming: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return http.StatusInternalServerError
	}
}
