package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/database"
	"github.com/video-storage-api/internal/routes"
	"github.com/video-storage-api/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	db, err := database.New(dbCtx, cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("database connected")

	store, err := storage.New(cfg.Storage)
	if err != nil {
		log.Fatalf("failed to initialise storage: %v", err)
	}
	log.Printf("storage ready: videos=%s thumbnails=%s", store.VideoDir(), store.ThumbnailDir())

	router := routes.Register(db, store, cfg)

	// Note: ReadTimeout/WriteTimeout are 0 (no limit) so large uploads
	// have time to complete. We still guard against slow-loris attacks
	// via ReadHeaderTimeout, and against oversized bodies via per-route
	// http.MaxBytesReader.
	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("api server listening on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case sig := <-quit:
		log.Printf("received signal %s, shutting down gracefully", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("server exited cleanly")
}
