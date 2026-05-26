package jobs

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/video-storage-api/internal/config"
)

type CleanupRunner struct {
	cfg     config.CleanupConfig
	storage config.StorageConfig
}

func NewCleanupRunner(cfg config.CleanupConfig, storage config.StorageConfig) *CleanupRunner {
	return &CleanupRunner{cfg: cfg, storage: storage}
}

func (r *CleanupRunner) Run(ctx context.Context) {
	if !r.cfg.Enabled {
		log.Printf("[JOB] name=cleanup status=disabled")
		return
	}
	interval := r.cfg.Interval
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	r.runOnce()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[JOB] name=cleanup status=stopped")
			return
		case <-ticker.C:
			r.runOnce()
		}
	}
}

func (r *CleanupRunner) runOnce() {
	start := time.Now()
	removed := 0
	removed += removeOld(filepath.Join(r.storage.HLSPath, "tmp"), r.cfg.TempHLSMaxAge)
	removed += removeOld(filepath.Join(r.storage.HLSPath, ".tmp"), r.cfg.TempHLSMaxAge)
	removed += removeOld(filepath.Join(r.storage.HLSPath, "failed"), r.cfg.ArtifactMaxAge)
	removed += removeOld("/app/logs", r.cfg.LogMaxAge)
	log.Printf("[JOB] name=cleanup status=ok removed=%d took=%s", removed, time.Since(start))
}

func removeOld(root string, maxAge time.Duration) int {
	if maxAge <= 0 {
		return 0
	}
	now := time.Now()
	removed := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if now.Sub(info.ModTime()) < maxAge {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".tmp") || strings.HasPrefix(name, "tmp") || strings.HasSuffix(name, ".log") || strings.Contains(path, "/failed/") {
			if err := os.RemoveAll(path); err == nil {
				removed++
			}
		}
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	return removed
}
