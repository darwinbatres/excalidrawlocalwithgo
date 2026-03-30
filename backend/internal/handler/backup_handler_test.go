package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

type backupHandlerEnv struct {
	handler *handler.BackupHandler
	repo    *mocks.MockBackupRepo
	audit   *mocks.MockAuditRepo
	s3      *mocks.MockObjectStorage
}

func newBackupHandlerEnv(t *testing.T) *backupHandlerEnv {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockBackupRepo(ctrl)
	audit := mocks.NewMockAuditRepo(ctrl)
	s3 := mocks.NewMockObjectStorage(ctrl)
	cfg := &config.Config{BackupS3Bucket: "test-backups"}
	svc := service.NewBackupService(repo, audit, s3, cfg, testutil.NopLogger())
	h := handler.NewBackupHandler(svc)
	return &backupHandlerEnv{handler: h, repo: repo, audit: audit, s3: s3}
}

func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestBackupHandler_List(t *testing.T) {
	env := newBackupHandlerEnv(t)

	env.repo.EXPECT().ListMetadata(gomock.Any(), gomock.Any(), gomock.Any()).Return([]models.BackupMetadata{
		{ID: "b-1", Status: "completed"},
	}, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backups", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].([]any)
	assert.Len(t, data, 1)
}

func TestBackupHandler_List_Unauthorized(t *testing.T) {
	env := newBackupHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backups", nil)
	// No user ID in context
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBackupHandler_Get(t *testing.T) {
	env := newBackupHandlerEnv(t)

	env.repo.EXPECT().GetMetadata(gomock.Any(), "b-1").Return(&models.BackupMetadata{
		ID:     "b-1",
		Status: "completed",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backups/b-1", nil)
	req = withUserID(req, "user-1")
	req = withChiURLParam(req, "id", "b-1")
	rec := httptest.NewRecorder()

	env.handler.Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBackupHandler_Download(t *testing.T) {
	env := newBackupHandlerEnv(t)

	env.repo.EXPECT().GetMetadata(gomock.Any(), "b-1").Return(&models.BackupMetadata{
		ID:         "b-1",
		Status:     "completed",
		StorageKey: "backups/test.sql",
	}, nil)
	env.s3.EXPECT().PresignedURLFromBucket(gomock.Any(), "test-backups", "backups/test.sql").
		Return("https://s3.example.com/download", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backups/b-1/download", nil)
	req = withUserID(req, "user-1")
	req = withChiURLParam(req, "id", "b-1")
	rec := httptest.NewRecorder()

	env.handler.Download(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].(map[string]any)
	assert.Contains(t, data["url"], "download")
}

func TestBackupHandler_Delete(t *testing.T) {
	env := newBackupHandlerEnv(t)

	env.repo.EXPECT().GetMetadata(gomock.Any(), "b-1").Return(&models.BackupMetadata{
		ID:         "b-1",
		Filename:   "backup.sql",
		StorageKey: "backups/test.sql",
	}, nil)
	env.s3.EXPECT().DeleteFromBucket(gomock.Any(), "test-backups", "backups/test.sql").Return(nil)
	env.repo.EXPECT().DeleteMetadata(gomock.Any(), "b-1").Return(nil)
	env.audit.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), models.AuditActionBackupDelete, "backup", "b-1", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/backups/b-1", nil)
	req = withUserID(req, "user-1")
	req = withChiURLParam(req, "id", "b-1")
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestBackupHandler_GetSchedule(t *testing.T) {
	env := newBackupHandlerEnv(t)

	env.repo.EXPECT().GetSchedule(gomock.Any()).Return(&models.BackupSchedule{
		Enabled:  true,
		CronExpr: "0 3 * * *",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backups/schedule", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.GetSchedule(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBackupHandler_UpdateSchedule(t *testing.T) {
	env := newBackupHandlerEnv(t)

	env.repo.EXPECT().UpdateSchedule(gomock.Any(), true, "0 3 * * *", 7, 4, 6).
		Return(&models.BackupSchedule{
			Enabled:     true,
			CronExpr:    "0 3 * * *",
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
		}, nil)
	env.audit.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), models.AuditActionBackupScheduleUpdate, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	body := `{"enabled":true,"cronExpr":"0 3 * * *","keepDaily":7,"keepWeekly":4,"keepMonthly":6}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/backups/schedule", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.UpdateSchedule(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
