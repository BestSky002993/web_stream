package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		reqID, _ := c.Get("request_id")

		log.Printf("[API] request_id=%v %s %s %d %s %s", reqID, method, path, status, latency, clientIP)
	}
}
