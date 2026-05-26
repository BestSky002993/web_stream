// Package categories owns the admin category management feature.
//
// Layers (each in its own file):
//
//	category.go      domain model + status constants
//	slug.go          NormalizeSlug — testable, stateless
//	errors.go        typed service errors
//	dto.go           request/response shapes used by the HTTP layer
//	repository.go    SQL only; no business rules
//	service.go       validation, normalization, duplicate handling
//	handler.go       HTTP I/O; thin wrapper over the service
//	module.go        DI wiring + route registration
package categories
