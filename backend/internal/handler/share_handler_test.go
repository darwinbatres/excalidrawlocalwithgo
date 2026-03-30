package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type shareHandlerEnv struct {
	handler   *handler.ShareHandler
	shareRepo *mocks.MockShareLinkRepo
	boardRepo *mocks.MockBoardRepo
	memberRepo *mocks.MockMembershipRepo
	permRepo  *mocks.MockBoardPermissionRepo
	auditRepo *mocks.MockAuditRepo
}

func newShareHandlerEnv(t *testing.T) *shareHandlerEnv {
	ctrl := gomock.NewController(t)
	shareRepo := mocks.NewMockShareLinkRepo(ctrl)
	boardRepo := mocks.NewMockBoardRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	permRepo := mocks.NewMockBoardPermissionRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	accessSvc := service.NewAccessService(memberRepo)
	accessSvc.WithBoardRepos(boardRepo, permRepo)

	shareSvc := service.NewShareService(shareRepo, boardRepo, auditRepo, accessSvc, testutil.NopLogger())
	h := handler.NewShareHandler(shareSvc)

	return &shareHandlerEnv{
		handler:   h,
		shareRepo: shareRepo,
		boardRepo: boardRepo,
		memberRepo: memberRepo,
		permRepo:  permRepo,
		auditRepo: auditRepo,
	}
}

// --- Create ---

func TestShareHandler_Create_Success(t *testing.T) {
	env := newShareHandlerEnv(t)

	// RequireBoardEdit: owner check
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.shareRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&models.ShareLink{
		ID: "sl-new", BoardID: "board-1", Token: "generated-token", Role: models.BoardRoleViewer,
	}, nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	body := `{"role":"VIEWER","expiresInSeconds":3600}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/share", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&respBody))
	data := respBody["data"].(map[string]any)
	assert.NotEmpty(t, data["token"])
}

func TestShareHandler_Create_DefaultViewer(t *testing.T) {
	env := newShareHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.shareRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&models.ShareLink{
		ID: "sl-new", BoardID: "board-1", Token: "generated-token", Role: models.BoardRoleViewer,
	}, nil)
	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/share", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestShareHandler_Create_Unauthorized(t *testing.T) {
	env := newShareHandlerEnv(t)

	body := `{"role":"VIEWER"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/share", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestShareHandler_Create_NoBoardID(t *testing.T) {
	env := newShareHandlerEnv(t)

	body := `{"role":"VIEWER"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards//share", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- List ---

func TestShareHandler_List_Success(t *testing.T) {
	env := newShareHandlerEnv(t)

	// RequireBoardEdit: owner check
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.shareRepo.EXPECT().ListByBoard(gomock.Any(), "board-1").Return([]models.ShareLink{
		{ID: "sl-1", BoardID: "board-1", Token: "tok-1", Role: models.BoardRoleViewer, CreatedAt: time.Now()},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/share", nil)
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

func TestShareHandler_List_Unauthorized(t *testing.T) {
	env := newShareHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/share", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Revoke ---

func TestShareHandler_Revoke_Success(t *testing.T) {
	env := newShareHandlerEnv(t)

	env.shareRepo.EXPECT().GetByID(gomock.Any(), "sl-1").Return(&models.ShareLink{
		ID: "sl-1", BoardID: "board-1", CreatedBy: "user-1",
	}, nil)

	// RequireBoardEdit: owner check
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.shareRepo.EXPECT().Delete(gomock.Any(), "sl-1").Return(nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board-1/share/sl-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1", "linkId": "sl-1"})
	rec := httptest.NewRecorder()

	env.handler.Revoke(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestShareHandler_Revoke_Unauthorized(t *testing.T) {
	env := newShareHandlerEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board-1/share/sl-1", nil)
	req = withChiParams(req, map[string]string{"id": "board-1", "linkId": "sl-1"})
	rec := httptest.NewRecorder()

	env.handler.Revoke(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestShareHandler_Revoke_NoLinkID(t *testing.T) {
	env := newShareHandlerEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board-1/share/", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Revoke(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetShared ---

func TestShareHandler_GetShared_Success(t *testing.T) {
	env := newShareHandlerEnv(t)

	env.shareRepo.EXPECT().GetByToken(gomock.Any(), "valid-token").Return(&models.ShareLink{
		ID: "sl-1", BoardID: "board-1", Token: "valid-token", Role: models.BoardRoleViewer,
	}, nil)

	env.boardRepo.EXPECT().GetByIDWithVersion(gomock.Any(), "board-1").Return(&models.BoardWithVersion{
		Board: models.Board{ID: "board-1", Title: "Shared Board"},
	}, nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/share/valid-token", nil)
	req = withChiParams(req, map[string]string{"token": "valid-token"})
	rec := httptest.NewRecorder()

	env.handler.GetShared(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestShareHandler_GetShared_NoToken(t *testing.T) {
	env := newShareHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/share/", nil)
	rec := httptest.NewRecorder()

	env.handler.GetShared(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestShareHandler_GetShared_InvalidToken(t *testing.T) {
	env := newShareHandlerEnv(t)

	env.shareRepo.EXPECT().GetByToken(gomock.Any(), "bad-token").Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/share/bad-token", nil)
	req = withChiParams(req, map[string]string{"token": "bad-token"})
	rec := httptest.NewRecorder()

	env.handler.GetShared(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
