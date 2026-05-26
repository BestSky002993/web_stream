package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/middleware"
)

// Module bundles the auth layer's wired dependencies so the rest of the
// app gets a single object to mount.
type Module struct {
	Repo    *AdminUserRepository
	Tokens  *TokenManager
	Service *Service
	Handler *Handler
}

// NewModule constructs the full auth dependency graph from raw collaborators.
func NewModule(pool *pgxpool.Pool, cfg config.AuthConfig) *Module {
	repo := NewAdminUserRepository(pool)
	tokens := NewTokenManager(cfg.JWTSecret, cfg.JWTExpiresIn)
	svc := NewService(repo, tokens, cfg.LoginMaxAttempts, cfg.LoginLockDuration)
	handler := NewHandler(svc)
	return &Module{
		Repo:    repo,
		Tokens:  tokens,
		Service: svc,
		Handler: handler,
	}
}

// RegisterRoutes mounts the auth endpoints under the given /api/v1/admin
// group. Public routes go on the group directly; protected routes go on
// a sub-group guarded by RequireAdmin.
func (m *Module) RegisterRoutes(adminGroup *gin.RouterGroup) {
	loginLimiter := middleware.NewRateLimiter(20, time.Minute)
	adminGroup.POST("/auth/login", loginLimiter.Middleware(func(c *gin.Context) string {
		return "login:" + c.ClientIP()
	}), m.Handler.Login)

	protected := adminGroup.Group("")
	protected.Use(RequireAdmin(m.Service))
	{
		protected.GET("/me", m.Handler.Me)
	}
}
