// Package hls owns the HLS (HTTP Live Streaming) pipeline:
//
//   - converting uploaded MP4 files into segmented HLS via ffmpeg
//   - serving the master + variant manifests and the .ts segments
//     under signed URLs
//   - tracking conversion status (`videos.hls_status` /
//     `videos.hls_ready_at` / `videos.hls_error`)
//
// HLS runs alongside the existing progressive-MP4 streaming surface
// (Phase 6) — both can serve playback for the same video. Players
// pick whichever URL the admin or site endpoint hands them.
//
// Layers (each in its own file):
//
//	path.go        safe path construction, variant + segment whitelists
//	processor.go   ffmpeg invocation + master.m3u8 writer + cleanup
//	repository.go  hls_* column updates on the existing videos table
//	service.go     orchestration (kick off, status transitions, URL issuance)
//	handler.go     signed manifest + segment delivery (with URL rewriting)
//	errors.go      typed errors
//	module.go      DI + route registration
//
// Single-variant ("v1") in Phase 13; the structure (variant
// whitelist, master playlist writer, per-variant subdirs) is built so
// future phases can drop in 360p/720p/1080p re-encodes without
// touching the URL or signing layers.
package hls
