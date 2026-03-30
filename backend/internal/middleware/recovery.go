package middleware

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
)

// Recovery catches panics in handlers and returns a 500 error without leaking internals.
func Recovery(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					log.Error().
						Interface("panic", rvr).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("panic recovered")

					response.Err(w, r, apierror.ErrInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
