package categories

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler is the HTTP layer for /api/v1/admin/categories. It is intentionally
// thin: parse, delegate, render.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /api/v1/admin/categories.
func (h *Handler) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	cat, err := h.svc.Create(c.Request.Context(), CreateParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(cat))
}

// List handles GET /api/v1/admin/categories.
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.svc.List(c.Request.Context(), ListParams{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		Status: c.Query("status"),
		Sort:   c.Query("sort"),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, ListCategoriesResponse{
		Data: toResponseList(result.Items),
		Pagination: Pagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// Get handles GET /api/v1/admin/categories/:id.
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	cat, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(cat))
}

// Update handles PUT /api/v1/admin/categories/:id.
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	cat, err := h.svc.Update(c.Request.Context(), id, UpdateParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(cat))
}

// Delete handles DELETE /api/v1/admin/categories/:id.
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// writeServiceError maps typed errors to HTTP status + JSON envelope.
// Unknown errors are logged server-side and surfaced as a generic 500.
func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "category not found"})
	case errors.Is(err, ErrSlugTaken):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "category slug already exists"})
	case errors.Is(err, ErrInvalidSlug):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid slug"})
	case errors.Is(err, ErrInvalidName):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid name"})
	case errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
	default:
		log.Printf("categories: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
