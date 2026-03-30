package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
)

// CSRF validates Origin/Referer headers on state-changing requests.
// Defense-in-depth alongside SameSite cookies — blocks cross-origin
// POSTs even from browsers that don't fully support SameSite.
func CSRF(allowedOrigins string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, o := range strings.Split(allowedOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			if u, err := url.Parse(o); err == nil {
				allowed[strings.ToLower(u.Scheme+"://"+u.Host)] = true
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check state-changing methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				// Fall back to Referer if Origin is missing (some browsers strip Origin)
				if ref := r.Header.Get("Referer"); ref != "" {
					if u, err := url.Parse(ref); err == nil {
						origin = u.Scheme + "://" + u.Host
					}
				}
			}

			// If neither Origin nor Referer is present, reject.
			// Legitimate browser requests always include at least one.
			if origin == "" {
				response.Err(w, r, apierror.ErrForbidden.WithMessage("Missing origin header"))
				return
			}

			if !allowed[strings.ToLower(origin)] {
				response.Err(w, r, apierror.ErrForbidden.WithMessage("Cross-origin request rejected"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
