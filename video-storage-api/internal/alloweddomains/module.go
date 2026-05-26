package alloweddomains

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
	g := adminGroup.Group("/allowed-domains")
	{
		g.GET("", m.Handler.List)
		g.POST("", m.Handler.Create)
		g.GET("/:id", m.Handler.Get)
		g.PUT("/:id", m.Handler.Update)
		g.DELETE("/:id", m.Handler.Delete)
	}
}
