package hls

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/streaming"
	"github.com/video-storage-api/internal/videos"
)

// Module bundles the wired dependencies for the HLS feature.
type Module struct {
	Paths     *Paths
	Processor *Processor
	Repo      *Repository
	Service   *Service
	Handler   *Handler
}

// NewModule constructs the HLS dependency graph.
//
//   - cfg.Storage.HLSPath         — root directory for HLS outputs
//   - cfg.Storage.VideoPath       — where source MP4s live (read-only here)
//   - cfg.Streaming.PublicBaseURL — passed through to the handler for issuing URLs
//
// Reuses the existing `streaming.Signer` under the `"hls"` scope so
// the same secret protects both pipelines without cross-scope replay.
func NewModule(
	pool *pgxpool.Pool,
	videosRepo *videos.Repository,
	signer *streaming.Signer,
	prober ResolutionProber,
	storageCfg config.StorageConfig,
	hlsCfg config.HLSConfig,
	publicBaseURL string,
) (*Module, error) {
	paths, err := NewPaths(storageCfg.HLSPath)
	if err != nil {
		return nil, fmt.Errorf("hls: paths: %w", err)
	}

	processorCfg := DefaultProcessorConfig
	processorCfg.HLSSegmentSeconds = hlsCfg.SegmentSeconds
	processorCfg.X264Preset = hlsCfg.X264Preset
	processorCfg.ConvertTimeout = hlsCfg.ConvertTimeout
	processor := NewProcessor(paths, storageCfg.VideoPath, prober, processorCfg)
	repo := NewRepository(pool)
	svc := NewService(processor, repo, videosRepo, signer)
	handler := NewHandler(svc, HandlerConfig{PublicBaseURL: publicBaseURL})

	return &Module{
		Paths:     paths,
		Processor: processor,
		Repo:      repo,
		Service:   svc,
		Handler:   handler,
	}, nil
}

// RegisterPublicRoutes mounts the signed HLS surface under /api/v1.
//
// Route order: the static `/master.m3u8` and `/.../index.m3u8` segments
// come before the catch-all `:segment` so Gin resolves them first.
func (m *Module) RegisterPublicRoutes(v1 *gin.RouterGroup) {
	hls := v1.Group("/hls")
	{
		hls.GET("/:video_id/master.m3u8", m.Handler.ServeMaster)
		hls.HEAD("/:video_id/master.m3u8", m.Handler.ServeMaster)

		hls.GET("/:video_id/:variant/index.m3u8", m.Handler.ServeVariantIndex)
		hls.HEAD("/:video_id/:variant/index.m3u8", m.Handler.ServeVariantIndex)

		hls.GET("/:video_id/:variant/:segment", m.Handler.ServeSegment)
		hls.HEAD("/:video_id/:variant/:segment", m.Handler.ServeSegment)
	}
}

// RegisterAdminRoutes mounts POST /admin/videos/:id/hls-url on the
// protected admin group (caller has already attached the JWT middleware).
func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.POST("/videos/:id/hls-url", m.Handler.IssueAdminURL)
}

// RegisterSiteRoutes mounts POST /site/videos/:id/hls-url on the X-API-Key
// protected group (caller has already attached the sites middleware).
func (m *Module) RegisterSiteRoutes(siteGroup *gin.RouterGroup) {
	siteGroup.POST("/videos/:id/hls-url", m.Handler.IssueSiteURL)
}
