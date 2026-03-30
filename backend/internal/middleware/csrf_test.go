package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRF_GETPassesThrough(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET should pass through, got %d", w.Code)
	}
}

func TestCSRF_HEADPassesThrough(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodHead, "/api/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("HEAD should pass through, got %d", w.Code)
	}
}

func TestCSRF_OPTIONSPassesThrough(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("OPTIONS should pass through, got %d", w.Code)
	}
}

func TestCSRF_POSTWithValidOrigin(t *testing.T) {
	h := CSRF("http://example.com,https://app.example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST with valid Origin should pass, got %d", w.Code)
	}
}

func TestCSRF_POSTWithInvalidOrigin(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("POST with invalid Origin should be 403, got %d", w.Code)
	}
}

func TestCSRF_POSTWithNoOriginNoReferer(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("POST with no Origin/Referer should be 403, got %d", w.Code)
	}
}

func TestCSRF_POSTWithRefererFallback(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("Referer", "http://example.com/page")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("POST with valid Referer should pass, got %d", w.Code)
	}
}

func TestCSRF_PUTBlocked(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPut, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("PUT with invalid Origin should be 403, got %d", w.Code)
	}
}

func TestCSRF_DELETEBlocked(t *testing.T) {
	h := CSRF("http://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodDelete, "/api/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("DELETE with no Origin should be 403, got %d", w.Code)
	}
}
