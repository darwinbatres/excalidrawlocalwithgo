package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type versionHandlerEnv struct {
	handler    *handler.VersionHandler
	boardRepo  *mocks.MockBoardRepo
	versionRepo *mocks.MockBoardVersionRepo
	memberRepo *mocks.MockMembershipRepo
	permRepo   *mocks.MockBoardPermissionRepo
	auditRepo  *mocks.MockAuditRepo
}

func newVersionHandlerEnv(t *testing.T) *versionHandlerEnv {
	ctrl := gomock.NewController(t)
	boardRepo := mocks.NewMockBoardRepo(ctrl)
	versionRepo := mocks.NewMockBoardVersionRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	permRepo := mocks.NewMockBoardPermissionRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	accessSvc := service.NewAccessService(memberRepo)
	accessSvc.WithBoardRepos(boardRepo, permRepo)

	boardSvc := service.NewBoardService(nil, boardRepo, versionRepo, auditRepo, accessSvc, testutil.NopLogger(), 100)
	h := handler.NewVersionHandler(boardSvc)

	return &versionHandlerEnv{
		handler:    h,
		boardRepo:  boardRepo,
		versionRepo: versionRepo,
		memberRepo: memberRepo,
		permRepo:   permRepo,
		auditRepo:  auditRepo,
	}
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestVersionHandler_List_Success(t *testing.T) {
	env := newVersionHandlerEnv(t)

	// AccessService.RequireBoardView calls boardRepo.GetByID
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.versionRepo.EXPECT().ListByBoard(gomock.Any(), "board-1", 50, 0).Return(
		[]repository.BoardVersionWithCreator{
			{BoardVersionMeta: models.BoardVersionMeta{ID: "v-1", BoardID: "board-1", Version: 1, CreatedAt: time.Now()}, CreatedByEmail: "user@test.com"},
		}, int64(1), nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/versions", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].([]any)
	assert.Len(t, data, 1)
}

func TestVersionHandler_List_Unauthorized(t *testing.T) {
	env := newVersionHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/versions", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVersionHandler_List_NoBoardID(t *testing.T) {
	env := newVersionHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards//versions", nil)
	req = withUserID(req, "user-1")
	// No chi param for "id"
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVersionHandler_Get_Success(t *testing.T) {
	env := newVersionHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.versionRepo.EXPECT().GetByBoardAndVersion(gomock.Any(), "board-1", 3).Return(&models.BoardVersion{
		ID: "v-3", BoardID: "board-1", Version: 3, SceneJSON: json.RawMessage(`{"elements":[]}`),
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/versions/3", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "3"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestVersionHandler_Get_InvalidVersion(t *testing.T) {
	env := newVersionHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/versions/abc", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "abc"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVersionHandler_Get_ZeroVersion(t *testing.T) {
	env := newVersionHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/versions/0", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "0"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVersionHandler_Get_NotFound(t *testing.T) {
	env := newVersionHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.versionRepo.EXPECT().GetByBoardAndVersion(gomock.Any(), "board-1", 99).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/versions/99", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "99"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestVersionHandler_Restore_NilPool(t *testing.T) {
	// RestoreVersion → SaveVersion uses pool.Begin — since pool is nil in test env,
	// this panics. We verify the handler reaches the service layer correctly.
	env := newVersionHandlerEnv(t)

	env.versionRepo.EXPECT().GetByBoardAndVersion(gomock.Any(), "board-1", 2).Return(&models.BoardVersion{
		ID: "v-2", BoardID: "board-1", Version: 2,
		SceneJSON: json.RawMessage(`{"elements":[]}`),
	}, nil)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/versions/2/restore", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "2"})
	rec := httptest.NewRecorder()

	// Panics because pool is nil — confirms we reach SaveVersion
	assert.Panics(t, func() {
		env.handler.Restore(rec, req)
	})
}

func TestVersionHandler_Restore_Unauthorized(t *testing.T) {
	env := newVersionHandlerEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/versions/2/restore", nil)
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "2"})
	rec := httptest.NewRecorder()

	env.handler.Restore(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVersionHandler_Restore_InvalidVersion(t *testing.T) {
	env := newVersionHandlerEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/versions/abc/restore", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "version": "abc"})
	rec := httptest.NewRecorder()

	env.handler.Restore(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVersionHandler_Save_Unauthorized(t *testing.T) {
	env := newVersionHandlerEnv(t)

	body := `{"sceneJson":{"elements":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/versions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Save(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVersionHandler_Save_NoBoardID(t *testing.T) {
	env := newVersionHandlerEnv(t)

	body := `{"sceneJson":{"elements":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards//versions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Save(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVersionHandler_Save_MissingSceneJSON(t *testing.T) {
	env := newVersionHandlerEnv(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/versions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Save(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
