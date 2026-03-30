package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type boardHandlerEnv struct {
	handler    *handler.BoardHandler
	boardRepo  *mocks.MockBoardRepo
	versionRepo *mocks.MockBoardVersionRepo
	memberRepo *mocks.MockMembershipRepo
	permRepo   *mocks.MockBoardPermissionRepo
	auditRepo  *mocks.MockAuditRepo
}

func newBoardHandlerEnv(t *testing.T) *boardHandlerEnv {
	ctrl := gomock.NewController(t)
	boardRepo := mocks.NewMockBoardRepo(ctrl)
	versionRepo := mocks.NewMockBoardVersionRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	permRepo := mocks.NewMockBoardPermissionRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	accessSvc := service.NewAccessService(memberRepo)
	accessSvc.WithBoardRepos(boardRepo, permRepo)

	boardSvc := service.NewBoardService(nil, boardRepo, versionRepo, auditRepo, accessSvc, testutil.NopLogger(), 100)
	h := handler.NewBoardHandler(boardSvc)

	return &boardHandlerEnv{
		handler:    h,
		boardRepo:  boardRepo,
		versionRepo: versionRepo,
		memberRepo: memberRepo,
		permRepo:   permRepo,
		auditRepo:  auditRepo,
	}
}

// --- Get ---

func TestBoardHandler_Get_Success(t *testing.T) {
	env := newBoardHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1", Title: "Test Board",
	}, nil)

	env.boardRepo.EXPECT().GetByIDWithVersion(gomock.Any(), "board-1").Return(&models.BoardWithVersion{
		Board: models.Board{ID: "board-1", OrgID: "org-1", OwnerID: "user-1", Title: "Test Board"},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].(map[string]any)
	assert.Equal(t, "Test Board", data["title"])
}

func TestBoardHandler_Get_Unauthorized(t *testing.T) {
	env := newBoardHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBoardHandler_Get_NoBoardID(t *testing.T) {
	env := newBoardHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBoardHandler_Get_NotFound(t *testing.T) {
	env := newBoardHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-999").Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-999", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-999"})
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- List ---

func TestBoardHandler_List_Success(t *testing.T) {
	env := newBoardHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleViewer,
	}, nil)

	env.boardRepo.EXPECT().Search(gomock.Any(), gomock.Any()).Return(&repository.BoardSearchResult{
		Boards: []models.Board{
			{ID: "board-1", OrgID: "org-1", Title: "Board 1", CreatedAt: time.Now()},
		},
		Total: 1,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/boards", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].([]any)
	assert.Len(t, data, 1)
}

func TestBoardHandler_List_Unauthorized(t *testing.T) {
	env := newBoardHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/boards", nil)
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBoardHandler_List_NoOrgAccess(t *testing.T) {
	env := newBoardHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/boards", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Update ---

func TestBoardHandler_Update_Success(t *testing.T) {
	env := newBoardHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1", Title: "Old Title",
	}, nil)

	env.boardRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Audit log is async — allow optional call
	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	body := `{"title":"New Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/boards/board-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&respBody))
	data := respBody["data"].(map[string]any)
	assert.Equal(t, "New Title", data["title"])
}

func TestBoardHandler_Update_Unauthorized(t *testing.T) {
	env := newBoardHandlerEnv(t)

	body := `{"title":"New Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/boards/board-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Update(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBoardHandler_Update_Forbidden(t *testing.T) {
	env := newBoardHandlerEnv(t)

	// User is not owner
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "other-user",
	}, nil)
	// Not an org admin
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleViewer,
	}, nil)
	// No board-level permission
	env.permRepo.EXPECT().GetByBoardAndUser(gomock.Any(), "board-1", "user-1").Return(nil, pgx.ErrNoRows)

	body := `{"title":"New Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/boards/board-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Update(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- Delete ---

func TestBoardHandler_Delete_Success(t *testing.T) {
	env := newBoardHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.boardRepo.EXPECT().Delete(gomock.Any(), "board-1").Return(nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestBoardHandler_Delete_Unauthorized(t *testing.T) {
	env := newBoardHandlerEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board-1", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBoardHandler_Delete_Forbidden(t *testing.T) {
	env := newBoardHandlerEnv(t)

	// Not owner
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "other-user",
	}, nil)
	// Not org owner
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleMember,
	}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/boards/board-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- Create ---

func TestBoardHandler_Create_Unauthorized(t *testing.T) {
	env := newBoardHandlerEnv(t)

	body := `{"title":"My Board"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBoardHandler_Create_NoOrgID(t *testing.T) {
	env := newBoardHandlerEnv(t)

	body := `{"title":"My Board"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs//boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBoardHandler_Create_InvalidBody(t *testing.T) {
	env := newBoardHandlerEnv(t)

	body := `{"title":""}` // title is required, min=1
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
