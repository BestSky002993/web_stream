package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/jobs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("[WORKER] status=starting concurrency=%d max_retries=%d", cfg.Jobs.WorkerConcurrency, cfg.Jobs.MaxRetries)
	runner := jobs.NewCleanupRunner(cfg.Cleanup, cfg.Storage)
	runner.Run(ctx)
	log.Printf("[WORKER] status=stopped pid=%d", os.Getpid())
}
