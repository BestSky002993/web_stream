// Package videos owns the admin-side video management feature: upload,
// metadata CRUD, and soft delete. Public streaming and ffmpeg/HLS
// processing live elsewhere and are added in later phases.
//
// Files:
//
//	video.go        domain model + status/visibility constants
//	slug.go         slug normalization (kept local to keep package self-contained)
//	errors.go       typed service errors
//	dto.go          request/response shapes
//	repository.go   SQL only
//	service.go      validation, slug normalization, category checks
//	handler.go      multipart upload + JSON-style admin endpoints
//	module.go       DI wiring + route registration
package videos
