package sites

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler is the HTTP layer for /api/v1/admin/sites.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /api/v1/admin/sites. Returns the plain API key
// in the response — the only time it leaves the API.
func (h *Handler) Create(c *gin.Context) {
	var req CreateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	result, err := h.svc.Create(c.Request.Context(), CreateParams{
		Name:              req.Name,
		Domain:            req.Domain,
		Status:            req.Status,
		DailyRequestLimit: req.DailyRequestLimit,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponseWithKey(result.Site, result.APIKey))
}

// List handles GET /api/v1/admin/sites.
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.svc.List(c.Request.Context(), ListParams{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		Status: c.Query("status"),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, ListSitesResponse{
		Data: toResponseList(result.Items),
		Pagination: Pagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// Get handles GET /api/v1/admin/sites/:id.
func (h *Handler) Get(c *gin.Context) {
	s, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(s))
}

// Me handles GET /api/v1/site/me. It lets the WordPress plugin show the
// registered backend domain without exposing key material.
func (h *Handler) Me(c *gin.Context) {
	s, ok := SiteFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "api key required"})
		return
	}
	c.JSON(http.StatusOK, toResponse(s))
}

// Update handles PUT /api/v1/admin/sites/:id.
func (h *Handler) Update(c *gin.Context) {
	var req UpdateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	s, err := h.svc.Update(c.Request.Context(), c.Param("id"), UpdateParams{
		Name:              req.Name,
		Domain:            req.Domain,
		Status:            req.Status,
		DailyRequestLimit: req.DailyRequestLimit,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(s))
}

// Disable handles POST /api/v1/admin/sites/:id/disable.
func (h *Handler) Disable(c *gin.Context) {
	s, err := h.svc.Disable(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(s))
}

// RotateKey handles POST /api/v1/admin/sites/:id/rotate-key.
// Returns the new plain key once; the old key stops working immediately.
func (h *Handler) RotateKey(c *gin.Context) {
	result, err := h.svc.RotateAPIKey(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponseWithKey(result.Site, result.APIKey))
}

// Delete handles DELETE /api/v1/admin/sites/:id (soft delete).
func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "site not found"})
	case errors.Is(err, ErrDomainTaken):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "domain already in use"})
	case errors.Is(err, ErrInvalidName):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid name"})
	case errors.Is(err, ErrInvalidDomain):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid domain"})
	case errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
	case errors.Is(err, ErrInvalidLimit):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid daily_request_limit"})
	default:
		log.Printf("sites: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
