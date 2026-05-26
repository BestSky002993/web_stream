package routes

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/alloweddomains"
	"github.com/video-storage-api/internal/auth"
	"github.com/video-storage-api/internal/cache"
	"github.com/video-storage-api/internal/categories"
	"github.com/video-storage-api/internal/categorymappings"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/database"
	"github.com/video-storage-api/internal/health"
	"github.com/video-storage-api/internal/hls"
	"github.com/video-storage-api/internal/middleware"
	"github.com/video-storage-api/internal/publications"
	"github.com/video-storage-api/internal/sites"
	"github.com/video-storage-api/internal/storage"
	"github.com/video-storage-api/internal/streaming"
	"github.com/video-storage-api/internal/videos"
)

func Register(db *database.DB, store *storage.LocalStorage, cfg *config.Config) *gin.Engine {
	router := gin.New()
	allowedDomainsModule := alloweddomains.NewModule(db.Pool)
	cacheClient := cache.New(cache.Config{
		Enabled:  cfg.Redis.Enabled,
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Prefix:   cfg.Redis.Prefix,
		TTL:      cfg.Redis.TTL,
		TLS:      cfg.Redis.TLS,
	})

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	// CORS goes near the front of the chain so OPTIONS preflights are
	// answered with 204 before NoRoute returns 404 for them.
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowCredentials: cfg.CORS.AllowCredentials,
		DomainPolicy:     alloweddomains.NewPolicy("", allowedDomainsModule.Repo, alloweddomains.TypeCORS),
	}))
	router.Use(middleware.Logger())

	healthHandler := health.NewHandler(db)
	router.GET("/health", healthHandler.Check)

	authModule := auth.NewModule(db.Pool, cfg.Auth)
	categoriesModule := categories.NewModule(db.Pool)
	categoryMappingsModule := categorymappings.NewModule(db.Pool)
	videosModule := videos.NewModule(db.Pool, store, cfg.Storage, cfg.Streaming.PublicBaseURL)
	streamingModule := streaming.NewModule(videosModule.Repo, store, cfg.Streaming, allowedDomainsModule.Repo)
	sitesModule := sites.NewModule(db.Pool)
	sitesModule.Service.SetCache(cacheClient, cfg.Redis.TTL)
	publicationsModule := publications.NewModule(db.Pool)
	streamingModule.Service.SetPublications(publicationsModule.Repo)
	streamingModule.Service.SetCategoryMappings(categoryMappingsModule.Repo)
	streamingModule.Service.SetCache(cacheClient, cfg.Redis.TTL)

	hlsModule, err := hls.NewModule(
		db.Pool,
		videosModule.Repo,
		streamingModule.Signer,
		store, // LocalStorage satisfies hls.ResolutionProber via ProbeResolution
		cfg.Storage,
		cfg.HLS,
		cfg.Streaming.PublicBaseURL,
	)
	if err != nil {
		log.Fatalf("init hls module: %v", err)
	}
	hlsModule.Service.SetPublications(publicationsModule.Repo)
	hlsModule.Service.SetCache(cacheClient, cfg.Redis.TTL)

	// Tell the videos module about the HLS pipeline. It will call
	// hlsModule.Service.Start after every successful upload — the
	// service runs the actual ffmpeg in a goroutine, so the upload
	// response returns immediately with hls.status='processing'.
	videosModule.Service.SetHLSStarter(hlsModule.Service)

	v1 := router.Group("/api/v1")
	{
		v1.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "video-storage-api",
				"version": "v1",
				"status":  "ok",
			})
		})

		// Public streaming surface (signed URLs / public videos / metadata / thumbnails).
		streamingModule.RegisterPublicRoutes(v1)
		// Public HLS surface (signed manifests + segments).
		hlsModule.RegisterPublicRoutes(v1)

		// /admin — JWT-protected (admin login lives here too, public).
		adminGroup := v1.Group("/admin")
		authModule.RegisterRoutes(adminGroup)

		protected := adminGroup.Group("")
		protected.Use(auth.RequireAdmin(authModule.Service))
		{
			categoriesModule.RegisterRoutes(protected)
			videosModule.RegisterRoutes(protected)
			streamingModule.RegisterAdminRoutes(protected)
			sitesModule.RegisterAdminRoutes(protected)
			allowedDomainsModule.RegisterAdminRoutes(protected)
			hlsModule.RegisterAdminRoutes(protected)
			categoryMappingsModule.RegisterAdminRoutes(protected)
			publicationsModule.RegisterAdminRoutes(protected)
		}

		// /site — X-API-Key-protected. Each request carries an authenticated
		// site in context.
		siteGroup := v1.Group("/site")
		siteGroup.Use(sitesModule.Middleware())
		siteLimiter := middleware.NewRateLimiter(240, time.Minute)
		siteGroup.Use(siteLimiter.Middleware(func(c *gin.Context) string {
			if site, ok := sites.SiteFromContext(c); ok {
				return "site:" + site.ID + ":" + c.ClientIP()
			}
			return "site:" + c.ClientIP()
		}))
		{
			sitesModule.RegisterSiteRoutes(siteGroup)
			streamingModule.RegisterSiteRoutes(siteGroup)
			hlsModule.RegisterSiteRoutes(siteGroup)
			categoryMappingsModule.RegisterSiteRoutes(siteGroup)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})

	return router
}
