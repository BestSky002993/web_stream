package sites

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module bundles the wired dependencies for the sites feature.
type Module struct {
	Repo    *Repository
	Service *Service
	Handler *Handler
}

// NewModule constructs the full dependency graph for sites.
func NewModule(pool *pgxpool.Pool) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	handler := NewHandler(svc)
	return &Module{Repo: repo, Service: svc, Handler: handler}
}

// RegisterAdminRoutes mounts the admin CRUD endpoints under the given
// protected admin group. The caller must have already attached the
// admin JWT middleware.
func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	g := adminGroup.Group("/sites")
	{
		g.POST("", m.Handler.Create)
		g.GET("", m.Handler.List)
		g.GET("/:id", m.Handler.Get)
		g.PUT("/:id", m.Handler.Update)
		g.POST("/:id/disable", m.Handler.Disable)
		g.POST("/:id/rotate-key", m.Handler.RotateKey)
		g.DELETE("/:id", m.Handler.Delete)
	}
}

// Middleware returns the X-API-Key middleware to mount on site-protected groups.
func (m *Module) Middleware() gin.HandlerFunc {
	return RequireSite(m.Service)
}

func (m *Module) RegisterSiteRoutes(siteGroup *gin.RouterGroup) {
	siteGroup.GET("/me", m.Handler.Me)
}
