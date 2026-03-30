package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/config"
	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/jwt"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type oauthHandlerEnv struct {
	handler     *handler.OAuthHandler
	userRepo    *mocks.MockUserRepo
	accountRepo *mocks.MockAccountRepo
	refreshRepo *mocks.MockRefreshTokenRepo
	auditRepo   *mocks.MockAuditRepo
}

func newOAuthHandlerEnv(t *testing.T) *oauthHandlerEnv {
	ctrl := gomock.NewController(t)
	userRepo := mocks.NewMockUserRepo(ctrl)
	accountRepo := mocks.NewMockAccountRepo(ctrl)
	refreshRepo := mocks.NewMockRefreshTokenRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	jwtMgr := jwt.NewManager("test-secret-key-32chars-minimum!", 15*time.Minute, 720*time.Hour)

	// No providers configured — tests for error paths
	cfg := &config.Config{Env: "development"}
	oauthSvc := service.NewOAuthService(cfg, userRepo, accountRepo, refreshRepo, auditRepo, jwtMgr, testutil.NopLogger())
	h := handler.NewOAuthHandler(oauthSvc)

	return &oauthHandlerEnv{
		handler:     h,
		userRepo:    userRepo,
		accountRepo: accountRepo,
		refreshRepo: refreshRepo,
		auditRepo:   auditRepo,
	}
}

// --- Authorize ---

func TestOAuthHandler_Authorize_InvalidProvider(t *testing.T) {
	env := newOAuthHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/unknown", nil)
	req = withChiParams(req, map[string]string{"provider": "unknown"})
	rec := httptest.NewRecorder()

	env.handler.Authorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOAuthHandler_Authorize_NoProviderParam(t *testing.T) {
	env := newOAuthHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/", nil)
	rec := httptest.NewRecorder()

	env.handler.Authorize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Callback ---

func TestOAuthHandler_Callback_MissingStateCookie(t *testing.T) {
	env := newOAuthHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback?code=abc&state=xyz", nil)
	req = withChiParams(req, map[string]string{"provider": "google"})
	rec := httptest.NewRecorder()

	env.handler.Callback(rec, req)

	// Missing oauth_state cookie
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOAuthHandler_Callback_StateMismatch(t *testing.T) {
	env := newOAuthHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct-state"})
	req = withChiParams(req, map[string]string{"provider": "google"})
	rec := httptest.NewRecorder()

	env.handler.Callback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOAuthHandler_Callback_ProviderError(t *testing.T) {
	env := newOAuthHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback?error=access_denied&error_description=user+denied&state=valid-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "valid-state"})
	req = withChiParams(req, map[string]string{"provider": "google"})
	rec := httptest.NewRecorder()

	env.handler.Callback(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestOAuthHandler_Callback_MissingCode(t *testing.T) {
	env := newOAuthHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback?state=valid-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "valid-state"})
	req = withChiParams(req, map[string]string{"provider": "google"})
	rec := httptest.NewRecorder()

	env.handler.Callback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
