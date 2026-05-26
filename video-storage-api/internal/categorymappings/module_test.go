package categorymappings

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesDoNotConflictAndKeepSiteScopedPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.GET("/sites/:id", func(c *gin.Context) {})
	site := router.Group("/api/v1/site")

	module := NewModule(nil)
	module.RegisterAdminRoutes(admin)
	module.RegisterSiteRoutes(site)

	want := map[string]bool{
		"GET /api/v1/admin/sites/:id/remote-categories":          false,
		"GET /api/v1/admin/sites/:id/category-mappings":          false,
		"PUT /api/v1/admin/sites/:id/category-mappings":          false,
		"POST /api/v1/admin/sites/:id/category-mappings/suggest": false,
		"POST /api/v1/site/sync/categories/import":               false,
		"GET /api/v1/site/sync/category-mappings":                false,
		"PUT /api/v1/site/sync/category-mappings":                false,
		"PUT /api/v1/site/sync/category-mappings/:id":            false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("route %s was not registered", route)
		}
	}
}
