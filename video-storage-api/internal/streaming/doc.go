// Package streaming serves video bytes from private storage over HTTP
// in a controlled way.
//
// Endpoints:
//
//	GET  /api/v1/stream/:videoID                  — Range-aware streaming
//	GET  /api/v1/videos/:slug                     — public metadata
//	GET  /api/v1/thumbnails/:videoID              — auto-generated thumbnail
//	POST /api/v1/admin/videos/:id/stream-url      — admin signs a stream URL
//
// Security layers (top to bottom):
//
//  1. Domain whitelist (Origin / Referer) — hotlink protection
//  2. Signed URL check — every MP4 byte response requires a valid
//     HMAC-signed URL with a future `expires`
//  3. Status check — only `status='ready'` videos are streamable
//  4. Strict filename sanitisation before any disk I/O — no client-
//     supplied path component ever reaches the filesystem
//
// Layers (each in its own file):
//
//	signing.go    HMAC-SHA256 signed URLs
//	range.go      HTTP Range header parsing
//	domain.go     Origin/Referer host whitelist
//	mime.go       extension → response Content-Type
//	errors.go     typed service errors
//	service.go    orchestration + authz + file lookup
//	handler.go    HTTP I/O + Range writer + structured access log
//	module.go     DI wiring + route registration
package streaming
