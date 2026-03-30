package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/config"
	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/jwt"
	"github.com/darwinbatres/drawgo/backend/internal/realtime"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type wsHandlerEnv struct {
	handler    *handler.WSHandler
	hub        *realtime.Hub
	jwtMgr     *jwt.Manager
	shareRepo  *mocks.MockShareLinkRepo
	boardRepo  *mocks.MockBoardRepo
	memberRepo *mocks.MockMembershipRepo
	permRepo   *mocks.MockBoardPermissionRepo
}

func newWSHandlerEnv(t *testing.T) *wsHandlerEnv {
	ctrl := gomock.NewController(t)
	shareRepo := mocks.NewMockShareLinkRepo(ctrl)
	boardRepo := mocks.NewMockBoardRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	permRepo := mocks.NewMockBoardPermissionRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	log := testutil.NopLogger()

	jwtMgr := jwt.NewManager("test-secret-key-32chars-minimum!", 15*time.Minute, 720*time.Hour)

	accessSvc := service.NewAccessService(memberRepo)
	accessSvc.WithBoardRepos(boardRepo, permRepo)

	shareSvc := service.NewShareService(shareRepo, boardRepo, auditRepo, accessSvc, log)

	cfg := &config.Config{
		WSMaxConnsPerBoard:    50,
		WSMaxMessageSize:      1048576,
		WSWriteTimeout:        10 * time.Second,
		WSCursorBatchInterval: 33 * time.Millisecond,
		WSHeartbeatInterval:   30 * time.Second,
	}

	hub := realtime.NewHub(realtime.HubConfig{
		MaxConnsPerBoard:  cfg.WSMaxConnsPerBoard,
		HeartbeatInterval: cfg.WSHeartbeatInterval,
		CursorInterval:    cfg.WSCursorBatchInterval,
	}, log)

	h := handler.NewWSHandler(hub, jwtMgr, shareSvc, accessSvc, log, []string{"*"}, cfg)

	return &wsHandlerEnv{
		handler:    h,
		hub:        hub,
		jwtMgr:     jwtMgr,
		shareRepo:  shareRepo,
		boardRepo:  boardRepo,
		memberRepo: memberRepo,
		permRepo:   permRepo,
	}
}

// --- Stats ---

func TestWSHandler_Stats_Success(t *testing.T) {
	env := newWSHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/stats", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Stats(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].(map[string]any)
	assert.Contains(t, data, "activeRooms")
	assert.Contains(t, data, "totalClients")
}

func TestWSHandler_Stats_Unauthorized(t *testing.T) {
	env := newWSHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/stats", nil)
	rec := httptest.NewRecorder()

	env.handler.Stats(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Upgrade ---

func TestWSHandler_Upgrade_NoBoardID(t *testing.T) {
	env := newWSHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/boards/", nil)
	rec := httptest.NewRecorder()

	env.handler.Upgrade(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWSHandler_Upgrade_NoAuth(t *testing.T) {
	env := newWSHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/boards/board-1", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Upgrade(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWSHandler_Upgrade_InvalidJWT(t *testing.T) {
	env := newWSHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/boards/board-1?token=invalid-jwt-token", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Upgrade(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWSHandler_Upgrade_InvalidShareToken(t *testing.T) {
	env := newWSHandlerEnv(t)

	env.shareRepo.EXPECT().GetByToken(gomock.Any(), "bad-share").Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws/boards/board-1?share=bad-share", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Upgrade(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
