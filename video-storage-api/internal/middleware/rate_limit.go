package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	windowStart time.Time
	count       int
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
	limit   int
	window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets: map[string]rateBucket{},
		limit:   limit,
		window:  window,
	}
}

func (r *RateLimiter) Middleware(keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFn(c)
		if key == "" {
			key = c.ClientIP()
		}
		if !r.allow(key, time.Now()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func (r *RateLimiter) allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	b := r.buckets[key]
	if b.windowStart.IsZero() || now.Sub(b.windowStart) >= r.window {
		r.buckets[key] = rateBucket{windowStart: now, count: 1}
		return true
	}
	if b.count >= r.limit {
		return false
	}
	b.count++
	r.buckets[key] = b
	return true
}
