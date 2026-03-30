package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/jwt"
)

func TestAuth_ValidToken_Cookie(t *testing.T) {
	mgr := jwt.NewManager("test-secret", 15*time.Minute, 24*time.Hour)
	token, err := mgr.CreateAccessToken("user-123", "test@example.com")
	assert.NoError(t, err)

	var capturedUserID, capturedEmail string
	handler := middleware.Auth(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = middleware.UserIDFromCtx(r.Context())
		capturedEmail = middleware.EmailFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "__Host-access_token", Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-123", capturedUserID)
	assert.Equal(t, "test@example.com", capturedEmail)
}

func TestAuth_ValidToken_BearerHeader(t *testing.T) {
	mgr := jwt.NewManager("test-secret", 15*time.Minute, 24*time.Hour)
	token, err := mgr.CreateAccessToken("user-456", "bearer@example.com")
	assert.NoError(t, err)

	handler := middleware.Auth(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user-456", middleware.UserIDFromCtx(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_MissingToken(t *testing.T) {
	mgr := jwt.NewManager("test-secret", 15*time.Minute, 24*time.Hour)

	handler := middleware.Auth(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	mgr := jwt.NewManager("test-secret", 15*time.Minute, 24*time.Hour)

	handler := middleware.Auth(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_TamperedToken(t *testing.T) {
	mgr := jwt.NewManager("test-secret", 15*time.Minute, 24*time.Hour)
	differentMgr := jwt.NewManager("different-secret", 15*time.Minute, 24*time.Hour)
	token, err := differentMgr.CreateAccessToken("attacker", "evil@example.com")
	assert.NoError(t, err)

	handler := middleware.Auth(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestUserIDFromCtx_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	assert.Equal(t, "", middleware.UserIDFromCtx(req.Context()))
}

func TestEmailFromCtx_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	assert.Equal(t, "", middleware.EmailFromCtx(req.Context()))
}
