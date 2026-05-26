package publications

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Repo    *Repository
	Service *Service
	Handler *Handler
}

func NewModule(pool *pgxpool.Pool) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	handler := NewHandler(svc)
	return &Module{Repo: repo, Service: svc, Handler: handler}
}

func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/sites/:id/publications", m.Handler.ListForSite)
	adminGroup.GET("/videos/:id/publications", m.Handler.ListForVideo)
	adminGroup.GET("/videos/:id/site-distribution", m.Handler.GetSiteDistribution)
	adminGroup.PUT("/videos/:id/site-distribution", m.Handler.SaveSiteDistribution)
	adminGroup.POST("/sites/:id/video-distribution/bulk", m.Handler.BulkSaveSiteDistribution)
	adminGroup.PUT("/sites/:id/videos/:video_id/publication", m.Handler.UpsertForSiteVideo)
	adminGroup.POST("/publications/:id/approve", m.Handler.Approve)
	adminGroup.POST("/publications/:id/disable", m.Handler.Disable)
	adminGroup.POST("/publications/:id/resync", m.Handler.Resync)
	adminGroup.DELETE("/publications/:id", m.Handler.Delete)
}
