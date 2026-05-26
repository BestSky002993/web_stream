package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/database"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

type Response struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	resp := Response{
		Status:   "ok",
		Database: "connected",
	}

	statusCode := http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		resp.Status = "degraded"
		resp.Database = "disconnected"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, resp)
}
