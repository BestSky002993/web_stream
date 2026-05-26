package publications

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterAdminRoutesUsesExistingSiteWildcardName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.GET("/sites/:id", func(c *gin.Context) {})

	module := NewModule(nil)
	module.RegisterAdminRoutes(admin)

	const want = "/api/v1/admin/sites/:id/videos/:video_id/publication"
	for _, route := range router.Routes() {
		if route.Method == "PUT" && route.Path == want {
			return
		}
	}

	t.Fatalf("route %s was not registered", want)
}
