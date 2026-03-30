package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessName(t *testing.T) {
	if got := AccessName(true); got != "__Host-access_token" {
		t.Errorf("AccessName(true) = %q, want %q", got, "__Host-access_token")
	}
	if got := AccessName(false); got != "access_token" {
		t.Errorf("AccessName(false) = %q, want %q", got, "access_token")
	}
}

func TestRefreshName(t *testing.T) {
	if got := RefreshName(true); got != "__Secure-refresh_token" {
		t.Errorf("RefreshName(true) = %q, want %q", got, "__Secure-refresh_token")
	}
	if got := RefreshName(false); got != "refresh_token" {
		t.Errorf("RefreshName(false) = %q, want %q", got, "refresh_token")
	}
}

func TestSetAndReadAccess(t *testing.T) {
	tests := []struct {
		name   string
		secure bool
	}{
		{"production", true},
		{"development", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetAccess(w, "tok123", tt.secure, 3600)
			req := &http.Request{Header: http.Header{}}
			for _, c := range w.Result().Cookies() {
				req.AddCookie(c)
			}
			if got := ReadAccess(req); got != "tok123" {
				t.Errorf("ReadAccess() = %q, want %q", got, "tok123")
			}
		})
	}
}

func TestSetAndReadRefresh(t *testing.T) {
	tests := []struct {
		name   string
		secure bool
	}{
		{"production", true},
		{"development", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetRefresh(w, "ref456", tt.secure, 7200)
			req := &http.Request{Header: http.Header{}}
			for _, c := range w.Result().Cookies() {
				req.AddCookie(c)
			}
			if got := ReadRefresh(req); got != "ref456" {
				t.Errorf("ReadRefresh() = %q, want %q", got, "ref456")
			}
		})
	}
}

func TestReadAccess_FallbackToUnprefixed(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "oldtok"})
	if got := ReadAccess(req); got != "oldtok" {
		t.Errorf("ReadAccess() fallback = %q, want %q", got, "oldtok")
	}
}

func TestReadRefresh_FallbackToUnprefixed(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "oldref"})
	if got := ReadRefresh(req); got != "oldref" {
		t.Errorf("ReadRefresh() fallback = %q, want %q", got, "oldref")
	}
}

func TestReadAccess_Empty(t *testing.T) {
	req := &http.Request{Header: http.Header{}}
	if got := ReadAccess(req); got != "" {
		t.Errorf("ReadAccess() with no cookies = %q, want empty", got)
	}
}

func TestClearAccess(t *testing.T) {
	w := httptest.NewRecorder()
	ClearAccess(w, true)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected at least one Set-Cookie header")
	}
	for _, c := range cookies {
		if c.MaxAge > 0 {
			t.Errorf("expected MaxAge <= 0 for clear, got %d", c.MaxAge)
		}
	}
}

func TestClearRefresh(t *testing.T) {
	w := httptest.NewRecorder()
	ClearRefresh(w, false)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected at least one Set-Cookie header")
	}
	for _, c := range cookies {
		if c.MaxAge > 0 {
			t.Errorf("expected MaxAge <= 0 for clear, got %d", c.MaxAge)
		}
	}
}

func TestAccessCookieAttributes(t *testing.T) {
	w := httptest.NewRecorder()
	SetAccess(w, "val", true, 3600)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if !c.HttpOnly {
		t.Error("expected HttpOnly = true")
	}
	if !c.Secure {
		t.Error("expected Secure = true")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("expected Path=/, got %q", c.Path)
	}
}
