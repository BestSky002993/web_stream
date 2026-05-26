package streaming

import "strings"

// contentTypeForExt returns the Content-Type to use when streaming a
// stored video file. Falls back to application/octet-stream so a
// surprising extension never leaks the wrong media type.
func contentTypeForExt(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}

// contentTypeForThumbnail mirrors contentTypeForExt for thumbnails.
// Thumbnails are always JPEG in this build (ffmpeg-generated) but the
// switch is here so future formats slot in cleanly.
func contentTypeForThumbnail(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// extOf returns the extension (without the leading dot) of a stored
// filename. Returns "" if there's no extension.
func extOf(filename string) string {
	i := strings.LastIndexByte(filename, '.')
	if i < 0 || i == len(filename)-1 {
		return ""
	}
	return filename[i+1:]
}
