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

	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/storage"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type auditHandlerEnv struct {
	handler    *handler.AuditHandler
	auditRepo  *mocks.MockAuditRepo
	memberRepo *mocks.MockMembershipRepo
	backupRepo *mocks.MockBackupRepo
	s3         *mocks.MockObjectStorage
}

func newAuditHandlerEnv(t *testing.T) *auditHandlerEnv {
	ctrl := gomock.NewController(t)
	auditRepo := mocks.NewMockAuditRepo(ctrl)
	assetRepo := mocks.NewMockBoardAssetRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	backupRepo := mocks.NewMockBackupRepo(ctrl)
	s3 := mocks.NewMockObjectStorage(ctrl)

	// Default AnyTimes for best-effort calls
	auditRepo.EXPECT().GlobalCRUDBreakdown(gomock.Any()).Return(&repository.CRUDBreakdown{}, nil).AnyTimes()
	backupRepo.EXPECT().ListMetadata(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, 0, nil).AnyTimes()
	backupRepo.EXPECT().GetSchedule(gomock.Any()).Return(&models.BackupSchedule{}, nil).AnyTimes()

	accessSvc := service.NewAccessService(memberRepo)
	auditSvc := service.NewAuditService(auditRepo, assetRepo, backupRepo, accessSvc, s3, testutil.NopLogger(), "test-bucket", "test-backup-bucket")
	h := handler.NewAuditHandler(auditSvc)

	return &auditHandlerEnv{handler: h, auditRepo: auditRepo, memberRepo: memberRepo, backupRepo: backupRepo, s3: s3}
}

func TestAuditHandler_SystemStats_Success(t *testing.T) {
	env := newAuditHandlerEnv(t)

	env.auditRepo.EXPECT().GetSystemStats(gomock.Any()).Return(&repository.SystemStatsResult{
		Counts: repository.SystemCounts{
			Users:  10,
			Boards: 5,
		},
		Database: repository.DatabaseStats{
			DatabaseSizeBytes: 1024000,
			CacheHitRatio:     0.99,
			Uptime:            "2 days",
		},
	}, nil)

	env.s3.EXPECT().BucketStats(gomock.Any(), "test-bucket").Return(&storage.BucketStatsResult{
		Bucket:      "test-bucket",
		ObjectCount: 42,
		TotalBytes:  2048000,
	}, nil)
	env.s3.EXPECT().BucketStats(gomock.Any(), "test-backup-bucket").Return(&storage.BucketStatsResult{
		Bucket:      "test-backup-bucket",
		ObjectCount: 10,
		TotalBytes:  512000,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.SystemStats(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].(map[string]any)

	counts := data["counts"].(map[string]any)
	assert.Equal(t, float64(10), counts["users"])
	assert.Equal(t, float64(5), counts["boards"])

	db := data["database"].(map[string]any)
	assert.Equal(t, float64(1024000), db["databaseSizeBytes"])
	assert.Equal(t, 0.99, db["cacheHitRatio"])
}

func TestAuditHandler_SystemStats_Unauthorized(t *testing.T) {
	env := newAuditHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	// No userID in context
	rec := httptest.NewRecorder()

	env.handler.SystemStats(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuditHandler_SystemStats_ServiceError(t *testing.T) {
	env := newAuditHandlerEnv(t)

	env.auditRepo.EXPECT().GetSystemStats(gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.SystemStats(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --------------- List handler tests ---------------

func TestAuditHandler_List_Success(t *testing.T) {
	env := newAuditHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "mem-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	now := time.Now()
	env.auditRepo.EXPECT().Query(gomock.Any(), gomock.Any()).Return(&repository.AuditQueryResult{
		Events: []models.AuditEvent{
			{ID: "evt-1", OrgID: "org-1", Action: "board.create", TargetType: "board", TargetID: "b-1", CreatedAt: now},
			{ID: "evt-2", OrgID: "org-1", Action: "board.update", TargetType: "board", TargetID: "b-1", CreatedAt: now},
		},
		Total: 2,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit?limit=50&offset=0", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	data := body["data"].([]any)
	assert.Len(t, data, 2)

	meta := body["meta"].(map[string]any)
	assert.Equal(t, float64(2), meta["total"])
}

func TestAuditHandler_List_Unauthorized(t *testing.T) {
	env := newAuditHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit", nil)
	// No userID in context
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuditHandler_List_MissingOrgID(t *testing.T) {
	env := newAuditHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs//audit", nil)
	req = withUserID(req, "user-1")
	// No chi params → orgID is empty
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_Forbidden(t *testing.T) {
	env := newAuditHandlerEnv(t)

	// Return a MEMBER role which is below ADMIN
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "mem-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleMember,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuditHandler_List_ServiceError(t *testing.T) {
	env := newAuditHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "mem-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	env.auditRepo.EXPECT().Query(gomock.Any(), gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --------------- Stats handler tests ---------------

func TestAuditHandler_Stats_Success(t *testing.T) {
	env := newAuditHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "mem-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	env.auditRepo.EXPECT().Stats(gomock.Any(), "org-1", gomock.Any()).Return(&repository.AuditStats{
		TotalEvents: 100,
		ByAction: []repository.AuditActionCount{
			{Action: "board.create", Count: 60},
			{Action: "auth.login", Count: 40},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit/stats?days=30", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Stats(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(100), data["totalEvents"])
	byAction := data["byAction"].([]any)
	assert.Len(t, byAction, 2)
}

func TestAuditHandler_Stats_Unauthorized(t *testing.T) {
	env := newAuditHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit/stats", nil)
	// No userID in context
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Stats(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuditHandler_Stats_ServiceError(t *testing.T) {
	env := newAuditHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "mem-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	env.auditRepo.EXPECT().Stats(gomock.Any(), "org-1", gomock.Any()).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/audit/stats", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Stats(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
