package util

import "time"

// CachedLink defines a cached Link used to track Link execution time and state.
type CachedLink struct {
	State string
	Time  time.Time
	Sent  bool
}

// CacheContains checks if the LinkCache contains a Link ID.
func CacheContains(links map[string]CachedLink, link string) bool {
	_, p := links[link]
	if p {
		return true
	}
	return false
}
