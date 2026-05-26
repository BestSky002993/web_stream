package streaming

import "time"

// BuildSeedKey turns a caller-supplied seed string into the canonical
// key used by the database to order random results.
//
// The key is `<seed>:<UTC date>`, so the same seed produces a stable
// order for one calendar day (UTC) and a different order on the next.
// This lets each WordPress site pick a unique seed and get its own
// daily-rotating feed.
//
// An empty seed returns "" — the caller should treat that as "fresh
// random on every call".
func BuildSeedKey(seed string, now time.Time) string {
	if seed == "" {
		return ""
	}
	return seed + ":" + now.UTC().Format("2006-01-02")
}

// ClampLimit returns a usable page size given a caller request, a
// default, and a hard ceiling. Negative or zero requested → default.
func ClampLimit(requested, def, max int) int {
	if requested < 1 {
		return def
	}
	if requested > max {
		return max
	}
	return requested
}
