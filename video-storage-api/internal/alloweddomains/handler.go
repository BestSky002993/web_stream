package alloweddomains

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	d, err := h.svc.Create(c.Request.Context(), CreateParams{
		Domain: req.Domain,
		Type:   req.Type,
		Status: req.Status,
		Notes:  req.Notes,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toResponse(d))
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	result, err := h.svc.List(c.Request.Context(), ListParams{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		Type:   c.Query("type"),
		Status: c.Query("status"),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, ListResponse{
		Data: toResponseList(result.Items),
		Pagination: Pagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func (h *Handler) Get(c *gin.Context) {
	d, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(d))
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	d, err := h.svc.Update(c.Request.Context(), c.Param("id"), UpdateParams{
		Domain: req.Domain,
		Type:   req.Type,
		Status: req.Status,
		Notes:  req.Notes,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(d))
}

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
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "allowed domain not found"})
	case errors.Is(err, ErrDomainTaken):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "domain already exists"})
	case errors.Is(err, ErrInvalidDomain):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid domain"})
	case errors.Is(err, ErrInvalidType):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid type"})
	case errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
	default:
		log.Printf("allowed_domains: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
