package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
)

func TestSecurity_SetsAllHeaders(t *testing.T) {
	handler := middleware.Security("https://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	h := rr.Header()
	assert.Contains(t, h.Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, h.Get("Content-Security-Policy"), "frame-ancestors 'none'")
	assert.Contains(t, h.Get("Content-Security-Policy"), "connect-src 'self' https://example.com wss://example.com")
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
	assert.Contains(t, h.Get("Permissions-Policy"), "camera=()")
	assert.Equal(t, "0", h.Get("X-XSS-Protection"))
	assert.Contains(t, h.Get("Strict-Transport-Security"), "max-age=31536000")
}

func TestSecurity_MultipleOrigins(t *testing.T) {
	handler := middleware.Security("https://a.com,https://b.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "https://a.com")
	assert.Contains(t, csp, "wss://a.com")
	assert.Contains(t, csp, "https://b.com")
	assert.Contains(t, csp, "wss://b.com")
}

func TestSecurity_EmptyOrigins(t *testing.T) {
	handler := middleware.Security("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "default-src 'self'")
}
