package publications

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListForSite(c *gin.Context) {
	h.list(c, c.Param("id"), "")
}

func (h *Handler) ListForVideo(c *gin.Context) {
	h.list(c, "", c.Param("id"))
}

func (h *Handler) GetSiteDistribution(c *gin.Context) {
	rows, err := h.svc.ListSiteDistribution(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, SiteDistributionResponse{
		VideoID: c.Param("id"),
		Data:    toDistributionResponse(rows),
	})
}

func (h *Handler) SaveSiteDistribution(c *gin.Context) {
	var req SiteDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	p, err := h.svc.SaveSiteDistribution(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		log.Printf("publications: site distribution save failed video_id=%s site_id=%s status=%s sync_mode=%s err=%v",
			c.Param("id"), req.SiteID, req.Status, req.SyncMode, err)
		writeError(c, err)
		return
	}
	rows, err := h.svc.ListSiteDistribution(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, ToResponse(p))
		return
	}
	for _, row := range toDistributionResponse(rows) {
		if row.SiteID == req.SiteID {
			c.JSON(http.StatusOK, row)
			return
		}
	}
	c.JSON(http.StatusOK, ToResponse(p))
}

func (h *Handler) BulkSaveSiteDistribution(c *gin.Context) {
	var req BulkDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	result, err := h.svc.BulkSaveSiteDistribution(c.Request.Context(), BulkDistributionParams{
		SiteID:           c.Param("id"),
		VideoIDs:         req.VideoIDs,
		Status:           req.Status,
		SyncMode:         req.SyncMode,
		UnmappedBehavior: req.UnmappedBehavior,
	})
	if err != nil {
		log.Printf("publications: bulk distribution failed site_id=%s status=%s sync_mode=%s count=%d err=%v",
			c.Param("id"), req.Status, req.SyncMode, len(req.VideoIDs), err)
		writeError(c, err)
		return
	}
	log.Printf("[SITE_AUDIT] event=bulk_distribution site_id=%s status=%s sync_mode=%s approved=%d updated=%d skipped=%d",
		c.Param("id"), req.Status, req.SyncMode, result.Approved, result.Updated, result.Skipped)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) list(c *gin.Context, siteID, videoID string) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	result, err := h.svc.List(c.Request.Context(), ListParams{
		SiteID:  siteID,
		VideoID: videoID,
		Status:  c.Query("status"),
		Page:    page,
		Limit:   limit,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ListPublicationsResponse{
		Data: ToResponseList(result.Items),
		Pagination: Pagination{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func (h *Handler) UpsertForSiteVideo(c *gin.Context) {
	var req UpsertPublicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	p, err := h.svc.Upsert(c.Request.Context(), UpsertInput{
		SiteID:          c.Param("id"),
		VideoID:         c.Param("video_id"),
		Status:          req.Status,
		SyncMode:        req.SyncMode,
		RemotePostID:    req.RemotePostID,
		RemoteSlug:      req.RemoteSlug,
		CategoryMapping: req.CategoryMapping,
		Tags:            req.Tags,
		Notes:           req.Notes,
		LastSyncedAt:    req.LastSyncedAt,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ToResponse(p))
}

func (h *Handler) Approve(c *gin.Context) { h.setStatus(c, StatusApproved) }
func (h *Handler) Disable(c *gin.Context) { h.setStatus(c, StatusDisabled) }
func (h *Handler) Resync(c *gin.Context)  { h.setStatus(c, StatusApproved) }

func (h *Handler) setStatus(c *gin.Context, status string) {
	p, err := h.svc.SetStatus(c.Request.Context(), c.Param("id"), status)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ToResponse(p))
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "publication not found"})
	case errors.Is(err, ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid publication status"})
	case errors.Is(err, ErrInvalidSyncMode):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid publication sync mode"})
	case errors.Is(err, ErrInvalidVideoID):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid video id"})
	case errors.Is(err, ErrInvalidUnmappedBehavior):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid unmapped behavior"})
	case errors.Is(err, ErrBlockedCategorySync):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "category mapping sync mode blocks publish"})
	case errors.Is(err, ErrMissingCategoryMapping):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "category mapping is missing or disabled"})
	default:
		log.Printf("publications: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}

func toDistributionResponse(rows []DistributionRow) []SiteDistributionRow {
	out := make([]SiteDistributionRow, len(rows))
	for i := range rows {
		row := rows[i]
		status := StatusDraft
		syncMode := "draft"
		effectiveSyncMode := row.MappingSyncMode
		var publicationID *string
		var updatedAt *time.Time
		var pubResp *PublicationResponse
		if row.Publication != nil {
			status = distributionStatus(row.Publication.Status)
			if row.Publication.SyncMode != nil {
				syncMode = *row.Publication.SyncMode
				effectiveSyncMode = *row.Publication.SyncMode
			} else {
				syncMode = row.MappingSyncMode
			}
			publicationID = &row.Publication.ID
			updatedAt = &row.Publication.UpdatedAt
			resp := ToResponse(row.Publication)
			pubResp = &resp
		}
		warnings := []string{}
		if row.MappingStatus == "unmapped" {
			warnings = append(warnings, "Category is not mapped for this site.")
		}
		if row.MappingStatus == "disabled" {
			warnings = append(warnings, "Category mapping is disabled for this site.")
		}
		if row.MappingStatus == "unconfirmed" {
			warnings = append(warnings, "Category mapping is not confirmed by both WordPress and Stream.")
		}
		if row.MappingSyncMode == "block" {
			warnings = append(warnings, "Category mapping sync mode blocks publishing.")
		}
		out[i] = SiteDistributionRow{
			SiteID:             row.SiteID,
			SiteName:           row.SiteName,
			SiteDomain:         row.SiteDomain,
			SiteStatus:         row.SiteStatus,
			PublicationID:      publicationID,
			Status:             status,
			SyncMode:           syncMode,
			EffectiveSyncMode:  effectiveSyncMode,
			MappingStatus:      row.MappingStatus,
			MappingSyncMode:    row.MappingSyncMode,
			MappingEnabled:     row.MappingEnabled,
			RemoteCategoryID:   row.RemoteCategoryID,
			RemoteCategoryName: row.RemoteCategoryName,
			Warnings:           warnings,
			UpdatedAt:          updatedAt,
			Publication:        pubResp,
		}
	}
	return out
}

func distributionStatus(status string) string {
	switch status {
	case StatusApproved, StatusSynced:
		return StatusApproved
	case StatusBlocked, StatusDisabled:
		return StatusBlocked
	default:
		return StatusDraft
	}
}
