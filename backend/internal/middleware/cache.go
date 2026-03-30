package middleware

import (
	"net/http"
)

// CacheControl sets Cache-Control headers for API responses.
// Prevents accidental caching of authenticated/mutable API data.
func CacheControl(directive string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", directive)
			next.ServeHTTP(w, r)
		})
	}
}

// Common cache directives.
const (
	// CacheNoStore prevents any caching — default for API responses.
	CacheNoStore = "no-store, no-cache, must-revalidate"
	// CachePrivateShort allows private caches for 60 seconds — useful for
	// read-heavy endpoints like board listings.
	CachePrivateShort = "private, max-age=60"
	// CachePublicImmutable is for truly immutable content (e.g. versioned assets).
	CachePublicImmutable = "public, max-age=31536000, immutable"
)

// ETagFromHeader implements conditional GET using a server-provided ETag.
// If the client sends If-None-Match matching the provided etag, responds
// with 304 Not Modified. Handlers must call SetETag to provide the etag
// before writing the response body.
func ETagFromHeader(w http.ResponseWriter, r *http.Request, etag string) bool {
	if etag == "" {
		return false
	}

	// Ensure proper ETag format (weak or strong)
	quoted := `"` + etag + `"`
	w.Header().Set("ETag", quoted)

	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == quoted || match == "W/"+quoted {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	return false
}
