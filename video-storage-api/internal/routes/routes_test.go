package routes

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/database"
	"github.com/video-storage-api/internal/storage"
)

func TestRegisterDoesNotPanicWithPublicationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	base := t.TempDir()
	storageCfg := config.StorageConfig{
		VideoPath:     filepath.Join(base, "videos"),
		ThumbnailPath: filepath.Join(base, "thumbnails"),
		HLSPath:       filepath.Join(base, "hls"),
	}
	store, err := storage.New(storageCfg)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}

	router := Register(&database.DB{}, store, &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:         "test-secret",
			JWTExpiresIn:      time.Hour,
			LoginMaxAttempts:  5,
			LoginLockDuration: time.Minute,
		},
		Storage: storageCfg,
		Streaming: config.StreamingConfig{
			SigningSecret: "test-stream-secret",
			URLExpiration: time.Hour,
		},
	})

	const want = "/api/v1/admin/sites/:id/videos/:video_id/publication"
	for _, route := range router.Routes() {
		if route.Method == "PUT" && route.Path == want {
			return
		}
	}

	t.Fatalf("route %s was not registered", want)
}
