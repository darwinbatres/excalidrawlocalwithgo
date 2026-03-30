package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Logger returns a chi-compatible middleware that logs all requests using zerolog.
func Logger(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return hlog.NewHandler(log)(
			hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
				hlog.FromRequest(r).Info().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Int("status", status).
					Int("size", size).
					Dur("duration", duration).
					Str("request_id", middleware.GetReqID(r.Context())).
					Msg("request")
			})(
				hlog.RemoteAddrHandler("ip")(
					hlog.UserAgentHandler("user_agent")(
						hlog.RequestIDHandler("req_id", "X-Request-Id")(
							next,
						),
					),
				),
			),
		)
	}
}
