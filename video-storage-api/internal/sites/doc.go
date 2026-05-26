// Package sites owns the multi-tenant "external website" access surface.
//
// Each row in the `sites` table represents a single external frontend
// (e.g. one WordPress install) that is allowed to call the video API
// with an X-API-Key header instead of an admin JWT.
//
// Files:
//
//	site.go         domain model + status constants
//	apikey.go       secure random key generation + SHA-256 verification
//	errors.go       typed service errors
//	dto.go          admin request/response shapes
//	repository.go   SQL only
//	service.go      validation, key lifecycle, authentication
//	handler.go      admin HTTP I/O (CRUD + rotate-key + disable)
//	middleware.go   RequireSite (X-API-Key + domain check + access log)
//	module.go       DI + route registration
package sites
