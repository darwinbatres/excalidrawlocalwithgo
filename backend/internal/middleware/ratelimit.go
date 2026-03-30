package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"

	"github.com/darwinbatres/drawgo/backend/internal/config"
)

// RateLimit returns a general API rate limiter.
func RateLimit(cfg *config.Config) func(http.Handler) http.Handler {
	return httprate.LimitByIP(cfg.RateLimitRequestsPerMin, time.Minute)
}

// AuthRateLimit returns a stricter rate limiter for authentication endpoints.
func AuthRateLimit(cfg *config.Config) func(http.Handler) http.Handler {
	return httprate.LimitByIP(cfg.RateLimitAuthPerMin, time.Minute)
}

// UploadRateLimit returns a rate limiter for file upload endpoints.
func UploadRateLimit(cfg *config.Config) func(http.Handler) http.Handler {
	return httprate.LimitByIP(cfg.RateLimitUploadPerMin, time.Minute)
}

// WSRateLimit returns a rate limiter for WebSocket connection upgrades.
func WSRateLimit(cfg *config.Config) func(http.Handler) http.Handler {
	return httprate.LimitByIP(cfg.RateLimitWSPerMin, time.Minute)
}
