package categorymappings

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
	adminGroup.GET("/sites/:id/remote-categories", m.Handler.AdminListRemoteCategories)
	adminGroup.GET("/sites/:id/category-mappings", m.Handler.AdminListMappings)
	adminGroup.PUT("/sites/:id/category-mappings", m.Handler.AdminUpsertMappings)
	adminGroup.POST("/sites/:id/category-mappings/suggest", m.Handler.AdminSuggestMappings)
}

func (m *Module) RegisterSiteRoutes(siteGroup *gin.RouterGroup) {
	siteGroup.POST("/sync/categories/import", m.Handler.SiteImportRemoteCategories)
	siteGroup.GET("/sync/category-mappings", m.Handler.SiteListMappings)
	siteGroup.PUT("/sync/category-mappings", m.Handler.SiteUpsertMapping)
	siteGroup.PUT("/sync/category-mappings/:id", m.Handler.SiteUpdateMapping)
}
