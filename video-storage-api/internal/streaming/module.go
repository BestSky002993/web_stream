package streaming

import (
	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/alloweddomains"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/storage"
	"github.com/video-storage-api/internal/videos"
)

// Module bundles the wired dependencies for the streaming feature.
type Module struct {
	Signer  *Signer
	Policy  *DomainPolicy
	Service *Service
	Handler *Handler
}

// NewModule constructs the full dependency graph for streaming.
// The streaming module reuses the existing videos.Repository for
// read-only DB access.
func NewModule(repo *videos.Repository, store *storage.LocalStorage, cfg config.StreamingConfig, domains alloweddomains.ActiveDomainLister) *Module {
	signer := NewSigner(cfg.SigningSecret, cfg.URLExpiration)
	policy := NewDomainPolicy(cfg.AllowedDomains, domains)
	svc := NewService(repo, store, signer, policy)
	handler := NewHandler(svc, HandlerConfig{PublicBaseURL: cfg.PublicBaseURL})

	return &Module{
		Signer:  signer,
		Policy:  policy,
		Service: svc,
		Handler: handler,
	}
}

// RegisterPublicRoutes mounts the unauthenticated streaming endpoints
// under the given /api/v1 group.
//
// Route order: register the static `/videos` and `/videos/random`
// BEFORE `/videos/:slug` so httprouter resolves the static paths first.
func (m *Module) RegisterPublicRoutes(v1 *gin.RouterGroup) {
	v1.GET("/stream/:id", m.Handler.Stream)
	v1.HEAD("/stream/:id", m.Handler.Stream)

	v1.GET("/videos", m.Handler.PublicList)
	v1.GET("/videos/random", m.Handler.PublicRandom)
	v1.GET("/videos/:slug", m.Handler.PublicMetadata)

	v1.GET("/thumbnails/:id", m.Handler.Thumbnail)
	v1.HEAD("/thumbnails/:id", m.Handler.Thumbnail)
}

// RegisterAdminRoutes mounts the admin-only stream-URL issuer on the
// protected admin group (caller has already attached the JWT middleware).
func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.POST("/videos/:id/stream-url", m.Handler.IssueSignedURL)
}

// RegisterSiteRoutes mounts the site (X-API-Key) endpoints. The caller
// must have already attached the sites.RequireSite middleware to the
// group so every handler runs with an authenticated site in context.
//
// Route order: static `/videos` and `/videos/random` before `/videos/:id`.
func (m *Module) RegisterSiteRoutes(siteGroup *gin.RouterGroup) {
	siteGroup.GET("/videos", m.Handler.SiteList)
	siteGroup.GET("/videos/random", m.Handler.SiteRandom)
	siteGroup.GET("/sync/videos", m.Handler.SiteSyncVideos)
	siteGroup.GET("/sync/categories", m.Handler.SiteSyncCategories)
	siteGroup.POST("/sync/videos/:id/status", m.Handler.SiteSyncStatus)
	siteGroup.POST("/videos/:id/stream-url", m.Handler.SiteIssueSignedURL)
}
