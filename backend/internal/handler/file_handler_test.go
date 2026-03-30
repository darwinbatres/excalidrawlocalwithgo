package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/config"
	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type fileHandlerEnv struct {
	handler   *handler.FileHandler
	assetRepo *mocks.MockBoardAssetRepo
	boardRepo *mocks.MockBoardRepo
	versionRepo *mocks.MockBoardVersionRepo
	memberRepo *mocks.MockMembershipRepo
	permRepo  *mocks.MockBoardPermissionRepo
	auditRepo *mocks.MockAuditRepo
	s3        *mocks.MockObjectStorage
}

func newFileHandlerEnv(t *testing.T) *fileHandlerEnv {
	ctrl := gomock.NewController(t)
	assetRepo := mocks.NewMockBoardAssetRepo(ctrl)
	boardRepo := mocks.NewMockBoardRepo(ctrl)
	versionRepo := mocks.NewMockBoardVersionRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	permRepo := mocks.NewMockBoardPermissionRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)
	s3 := mocks.NewMockObjectStorage(ctrl)

	accessSvc := service.NewAccessService(memberRepo)
	accessSvc.WithBoardRepos(boardRepo, permRepo)

	fileSvc := service.NewFileService(assetRepo, boardRepo, versionRepo, s3, accessSvc, auditRepo, testutil.NopLogger(), 25*1024*1024)
	cfg := &config.Config{MaxFileSize: 25 * 1024 * 1024}
	h := handler.NewFileHandler(fileSvc, cfg, testutil.NopLogger())

	return &fileHandlerEnv{
		handler:   h,
		assetRepo: assetRepo,
		boardRepo: boardRepo,
		versionRepo: versionRepo,
		memberRepo: memberRepo,
		permRepo:  permRepo,
		auditRepo: auditRepo,
		s3:        s3,
	}
}

// --- Upload ---

func TestFileHandler_Upload_Unauthorized(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards/board-1/files", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.Upload(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFileHandler_Upload_NoBoardID(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/boards//files", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Upload(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Download ---

func TestFileHandler_Download_Unauthorized(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/files/file-1", nil)
	req = withChiParams(req, map[string]string{"id": "board-1", "fileId": "file-1"})
	rec := httptest.NewRecorder()

	env.handler.Download(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFileHandler_Download_NoBoardID(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards//files/file-1", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Download(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListAssets ---

func TestFileHandler_ListAssets_Success(t *testing.T) {
	env := newFileHandlerEnv(t)

	// RequireBoardView
	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-1").Return(&models.Board{
		ID: "board-1", OrgID: "org-1", OwnerID: "user-1",
	}, nil)

	env.assetRepo.EXPECT().ListByBoard(gomock.Any(), "board-1").Return([]models.BoardAsset{
		{ID: "asset-1", BoardID: "board-1", FileID: "file-1", MimeType: "image/png", SizeBytes: 1024, CreatedAt: time.Now()},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/files", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.ListAssets(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].([]any)
	assert.Len(t, data, 1)
}

func TestFileHandler_ListAssets_Unauthorized(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/files", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.ListAssets(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFileHandler_ListAssets_NotFound(t *testing.T) {
	env := newFileHandlerEnv(t)

	env.boardRepo.EXPECT().GetByID(gomock.Any(), "board-999").Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-999/files", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "board-999"})
	rec := httptest.NewRecorder()

	env.handler.ListAssets(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- BoardStorage ---

func TestFileHandler_BoardStorage_Unauthorized(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boards/board-1/storage", nil)
	req = withChiParams(req, map[string]string{"id": "board-1"})
	rec := httptest.NewRecorder()

	env.handler.BoardStorage(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- OrgStorage ---

func TestFileHandler_OrgStorage_Unauthorized(t *testing.T) {
	env := newFileHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/storage", nil)
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.OrgStorage(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
