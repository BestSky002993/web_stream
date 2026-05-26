package videos

import (
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/auth"
	"github.com/video-storage-api/internal/urlutil"
)

// Handler is the HTTP layer for /api/v1/admin/videos. It only handles
// I/O — no business rules, no SQL, no shell-outs.
type Handler struct {
	svc           *Service
	maxBodyBytes  int64
	publicBaseURL string // empty → derived per-request from Host header
}

// NewHandler returns a Handler. maxBodyBytes is enforced via
// http.MaxBytesReader on the upload endpoints. publicBaseURL is used
// when serializing absolute thumbnail URLs into the response.
func NewHandler(svc *Service, maxBodyBytes int64, publicBaseURL string) *Handler {
	return &Handler{svc: svc, maxBodyBytes: maxBodyBytes, publicBaseURL: publicBaseURL}
}

// Create handles POST /api/v1/admin/videos (multipart/form-data).
func (h *Handler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBodyBytes)
	reqID := requestID(c)

	var req CreateVideoFormFields
	if err := c.ShouldBind(&req); err != nil {
		if isBodyTooLarge(err) {
			log.Printf("videos: upload rejected request_id=%s reason=body_too_large content_length=%d limit_bytes=%d", reqID, c.Request.ContentLength, h.maxBodyBytes)
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{Error: "video file exceeds size limit"})
			return
		}
		log.Printf("videos: upload rejected request_id=%s reason=invalid_form err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid upload form"})
		return
	}

	videoHeader, err := c.FormFile("video")
	if err != nil {
		log.Printf("videos: upload rejected request_id=%s reason=missing_video err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "video file is required"})
		return
	}

	thumbSec := DefaultThumbnailSecond
	if req.ThumbnailSecond != "" {
		n, err := strconv.Atoi(req.ThumbnailSecond)
		if err != nil {
			log.Printf("videos: upload rejected request_id=%s reason=invalid_thumbnail_second value=%q", reqID, req.ThumbnailSecond)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "thumbnail_second must be an integer"})
			return
		}
		thumbSec = n
	}

	admin, ok := auth.AdminFromContext(c)
	if !ok {
		log.Printf("videos: upload rejected request_id=%s reason=missing_admin_context", reqID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	log.Printf("videos: upload received request_id=%s admin_id=%s original_name=%q size=%d content_length=%d title=%q category_id=%s",
		reqID, admin.ID, videoHeader.Filename, videoHeader.Size, c.Request.ContentLength, req.Title, req.CategoryID)

	created, err := h.svc.Create(c.Request.Context(), CreateParams{
		Title:             req.Title,
		Slug:              req.Slug,
		Description:       req.Description,
		CategoryID:        req.CategoryID,
		Visibility:        req.Visibility,
		ThumbnailSecond:   thumbSec,
		UploadedByAdminID: admin.ID,
		Video:             multipartFile(videoHeader),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	log.Printf("videos: upload created request_id=%s video_id=%s stored_filename=%s size=%d mime=%s duration=%v thumbnail_second=%d",
		reqID, created.ID, created.VideoFilename, created.VideoSizeBytes, created.VideoMimeType, intPtrLog(created.DurationSeconds), created.ThumbnailSecond)

	c.JSON(http.StatusCreated, h.toResponse(c, created))
}

// List handles GET /api/v1/admin/videos.
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.svc.List(c.Request.Context(), ListParams{
		Page:       page,
		Limit:      limit,
		Search:     c.Query("search"),
		CategoryID: c.Query("category_id"),
		Status:     c.Query("status"),
		Visibility: c.Query("visibility"),
		NotApprovedSiteID: c.Query("not_approved_site_id"),
		Sort:       c.Query("sort"),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, ListVideosResponse{
		Data: h.toResponseList(c, result.Items),
		Pagination: Pagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// Get handles GET /api/v1/admin/videos/:id.
func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toResponse(c, v))
}

// Update handles PUT /api/v1/admin/videos/:id (multipart/form-data).
func (h *Handler) Update(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBodyBytes)

	var req UpdateVideoFormFields
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	params := UpdateParams{
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Status:      req.Status,
		Visibility:  req.Visibility,
	}

	if req.ThumbnailSecond != "" {
		n, err := strconv.Atoi(req.ThumbnailSecond)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "thumbnail_second must be an integer"})
			return
		}
		params.ThumbnailSecond = &n
	}

	v, err := h.svc.Update(c.Request.Context(), c.Param("id"), params)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toResponse(c, v))
}

// Delete handles DELETE /api/v1/admin/videos/:id (soft delete).
func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) toResponse(c *gin.Context, v *Video) VideoResponse {
	return toResponse(v, urlutil.ThumbnailURL(c, h.publicBaseURL, v.ID))
}

func (h *Handler) toResponseList(c *gin.Context, vs []Video) []VideoResponse {
	out := make([]VideoResponse, len(vs))
	for i := range vs {
		out[i] = h.toResponse(c, &vs[i])
	}
	return out
}

// multipartFile adapts a Gin/multipart file header into the service's
// UploadFile shape.
func multipartFile(fh *multipart.FileHeader) UploadFile {
	if fh == nil {
		return UploadFile{}
	}
	return UploadFile{
		OriginalName: fh.Filename,
		Size:         fh.Size,
		Open: func() (io.ReadCloser, error) {
			return fh.Open()
		},
	}
}

// writeServiceError maps typed errors to HTTP responses.
func writeServiceError(c *gin.Context, err error) {
	reqID := requestID(c)
	switch {
	case errors.Is(err, ErrNotFound):
		log.Printf("videos: request failed request_id=%s status=404 err=%v", reqID, err)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "video not found"})
	case errors.Is(err, ErrCategoryNotFound):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "category not found"})
	case errors.Is(err, ErrSlugTaken):
		log.Printf("videos: request failed request_id=%s status=409 err=%v", reqID, err)
		c.JSON(http.StatusConflict, ErrorResponse{Error: "video slug already exists"})
	case errors.Is(err, ErrInvalidSlug):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid slug"})
	case errors.Is(err, ErrInvalidTitle):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid title"})
	case errors.Is(err, ErrInvalidStatus):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
	case errors.Is(err, ErrInvalidVisibility):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid visibility"})
	case errors.Is(err, ErrInvalidSiteID):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid site id"})
	case errors.Is(err, ErrInvalidThumbnailSecond):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid thumbnail_second"})
	case errors.Is(err, ErrUnsupportedVideoType):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unsupported video format"})
	case errors.Is(err, ErrVideoTooLarge):
		log.Printf("videos: request failed request_id=%s status=413 err=%v", reqID, err)
		c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{Error: "video file exceeds size limit"})
	case errors.Is(err, ErrMissingVideo):
		log.Printf("videos: request failed request_id=%s status=400 err=%v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "video file is required"})
	case errors.Is(err, ErrThumbnailGeneration):
		log.Printf("videos: request failed request_id=%s status=422 err=%v", reqID, err)
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: "thumbnail generation failed; please try a lower thumbnail second or re-upload the video"})
	default:
		log.Printf("videos: internal error request_id=%s: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}

func isBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "http: request body too large")
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return c.Writer.Header().Get("X-Request-ID")
}

func intPtrLog(v *int) any {
	if v == nil {
		return "unknown"
	}
	return *v
}
