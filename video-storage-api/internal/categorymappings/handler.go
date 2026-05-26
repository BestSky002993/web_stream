package categorymappings

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/sites"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) AdminListRemoteCategories(c *gin.Context) {
	items, err := h.svc.ListRemoteCategories(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, RemoteCategoryListResponse{Data: ToRemoteCategoryResponseList(items)})
}

func (h *Handler) AdminListMappings(c *gin.Context) {
	h.listMappings(c, c.Param("id"))
}

func (h *Handler) AdminUpsertMappings(c *gin.Context) {
	var req MappingUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	items, err := h.svc.UpsertAdminMappings(c.Request.Context(), c.Param("id"), upsertItems(req.Mappings))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, MappingListResponse{Data: ToMappingResponseList(items)})
}

func (h *Handler) AdminSuggestMappings(c *gin.Context) {
	items, err := h.svc.SuggestSameNameMappings(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, MappingListResponse{Data: ToMappingResponseList(items)})
}

func (h *Handler) SiteImportRemoteCategories(c *gin.Context) {
	var req RemoteCategoryImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("categorymappings: invalid category import JSON: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body", Reason: "expected JSON object with categories array"})
		return
	}
	items := req.items()
	if len(items) == 0 {
		log.Printf("categorymappings: invalid category import: categories empty")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category", Field: "categories", Reason: "must contain at least one category"})
		return
	}
	imported, err := h.svc.ImportRemoteCategories(c.Request.Context(), siteIDFromContext(c), importItems(items))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, RemoteCategoryListResponse{Data: ToRemoteCategoryResponseList(imported)})
}

func (h *Handler) SiteListMappings(c *gin.Context) {
	h.listMappings(c, siteIDFromContext(c))
}

func (h *Handler) SiteUpsertMapping(c *gin.Context) {
	var req SiteMappingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if req.BackendCategoryID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "backend_category_id required"})
		return
	}
	h.updateSiteMapping(c, req.BackendCategoryID, req)
}

func (h *Handler) SiteUpdateMapping(c *gin.Context) {
	var req SiteMappingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	h.updateSiteMapping(c, c.Param("id"), req)
}

func (h *Handler) updateSiteMapping(c *gin.Context, mappingKey string, req SiteMappingUpdateRequest) {
	row, err := h.svc.UpdateSiteMapping(c.Request.Context(), siteIDFromContext(c), mappingKey, SiteMappingUpdateInput{
		RemoteCategoryID: req.RemoteCategoryID,
		Enabled:          req.Enabled,
		SyncMode:         req.SyncMode,
		ConfirmedByWP:    req.ConfirmedByWP,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ToMappingResponse(*row))
}

func (h *Handler) listMappings(c *gin.Context, siteID string) {
	items, err := h.svc.ListMappings(c.Request.Context(), siteID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, MappingListResponse{Data: ToMappingResponseList(items)})
}

func siteIDFromContext(c *gin.Context) string {
	if site, ok := sites.SiteFromContext(c); ok {
		return site.ID
	}
	return ""
}

func writeError(c *gin.Context, err error) {
	var validationErr *ValidationError
	switch {
	case errors.As(err, &validationErr):
		log.Printf("categorymappings: invalid category import: field=%s reason=%s", validationErr.Field, validationErr.Reason)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category", Field: validationErr.Field, Reason: validationErr.Reason})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "category mapping not found"})
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidSlug), errors.Is(err, ErrInvalidMode), errors.Is(err, ErrCrossSiteData):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	default:
		log.Printf("categorymappings: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
