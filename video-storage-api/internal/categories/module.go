package categories

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Module bundles the wired dependencies for the categories feature.
type Module struct {
	Repo    *Repository
	Service *Service
	Handler *Handler
}

// NewModule constructs the full categories dependency graph.
func NewModule(pool *pgxpool.Pool) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo)
	handler := NewHandler(svc)
	return &Module{
		Repo:    repo,
		Service: svc,
		Handler: handler,
	}
}

// RegisterRoutes mounts /categories under the given protected admin group.
// The caller is responsible for applying the JWT middleware to `group`
// before calling this.
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	cats := group.Group("/categories")
	{
		cats.POST("", m.Handler.Create)
		cats.GET("", m.Handler.List)
		cats.GET("/:id", m.Handler.Get)
		cats.PUT("/:id", m.Handler.Update)
		cats.DELETE("/:id", m.Handler.Delete)
	}
}
