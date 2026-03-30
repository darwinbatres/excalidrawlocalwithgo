package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheControl_SetsHeader(t *testing.T) {
	handler := CacheControl(CacheNoStore)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	got := w.Header().Get("Cache-Control")
	if got != CacheNoStore {
		t.Errorf("Cache-Control = %q, want %q", got, CacheNoStore)
	}
}

func TestCacheControl_PrivateShort(t *testing.T) {
	handler := CacheControl(CachePrivateShort)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	got := w.Header().Get("Cache-Control")
	if got != CachePrivateShort {
		t.Errorf("Cache-Control = %q, want %q", got, CachePrivateShort)
	}
}

func TestETagFromHeader_NotModified(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/boards/123", nil)
	req.Header.Set("If-None-Match", `"abc123"`)

	matched := ETagFromHeader(w, req, "abc123")

	if !matched {
		t.Error("expected ETagFromHeader to return true (match)")
	}
	if w.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %d", w.Code)
	}
}

func TestETagFromHeader_NoMatch(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/boards/123", nil)
	req.Header.Set("If-None-Match", `"old-etag"`)

	matched := ETagFromHeader(w, req, "new-etag")

	if matched {
		t.Error("expected ETagFromHeader to return false (no match)")
	}
	if got := w.Header().Get("ETag"); got != `"new-etag"` {
		t.Errorf("ETag header = %q, want %q", got, `"new-etag"`)
	}
}

func TestETagFromHeader_NoClientETag(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/boards/123", nil)

	matched := ETagFromHeader(w, req, "server-etag")

	if matched {
		t.Error("expected false when client has no If-None-Match")
	}
	if got := w.Header().Get("ETag"); got != `"server-etag"` {
		t.Errorf("ETag header = %q, want %q", got, `"server-etag"`)
	}
}

func TestETagFromHeader_EmptyETag(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/boards/123", nil)

	matched := ETagFromHeader(w, req, "")

	if matched {
		t.Error("expected false for empty etag")
	}
}

func TestETagFromHeader_WeakMatch(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/boards/123", nil)
	req.Header.Set("If-None-Match", `W/"abc123"`)

	matched := ETagFromHeader(w, req, "abc123")

	if !matched {
		t.Error("expected weak ETag match to succeed")
	}
	if w.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %d", w.Code)
	}
}
