package videos

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/storage"
)

// Module bundles the wired dependencies for the videos feature.
type Module struct {
	Repo    *Repository
	Service *Service
	Handler *Handler
}

// NewModule builds the full dependency graph for the videos feature.
// The caller is responsible for protecting the registered routes with
// the admin JWT middleware. `publicBaseURL` is used when serializing
// absolute thumbnail URLs into responses (empty → derive from request).
func NewModule(pool *pgxpool.Pool, store *storage.LocalStorage, cfg config.StorageConfig, publicBaseURL string) *Module {
	repo := NewRepository(pool)
	svc := NewService(repo, store, cfg.MaxVideoBytes)

	// Upload endpoint body cap = max video bytes + 1 MiB headroom for
	// the multipart envelope and text fields. Thumbnails are extracted
	// from the uploaded video, not received as a separate part.
	bodyCap := cfg.MaxVideoBytes + (1 << 20)

	return &Module{
		Repo:    repo,
		Service: svc,
		Handler: NewHandler(svc, bodyCap, publicBaseURL),
	}
}

// RegisterRoutes mounts /videos under the given protected admin group.
func (m *Module) RegisterRoutes(group *gin.RouterGroup) {
	vids := group.Group("/videos")
	{
		vids.POST("", m.Handler.Create)
		vids.GET("", m.Handler.List)
		vids.GET("/:id", m.Handler.Get)
		vids.PUT("/:id", m.Handler.Update)
		vids.DELETE("/:id", m.Handler.Delete)
	}
}
