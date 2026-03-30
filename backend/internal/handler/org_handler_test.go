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

type orgHandlerEnv struct {
	handler    *handler.OrgHandler
	orgRepo    *mocks.MockOrgRepo
	memberRepo *mocks.MockMembershipRepo
	userRepo   *mocks.MockUserRepo
	auditRepo  *mocks.MockAuditRepo
}

func newOrgHandlerEnv(t *testing.T) *orgHandlerEnv {
	ctrl := gomock.NewController(t)
	orgRepo := mocks.NewMockOrgRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	userRepo := mocks.NewMockUserRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	accessSvc := service.NewAccessService(memberRepo)

	orgSvc := service.NewOrgService(nil, orgRepo, memberRepo, userRepo, auditRepo, accessSvc, testutil.NopLogger())
	h := handler.NewOrgHandler(orgSvc)

	return &orgHandlerEnv{
		handler:    h,
		orgRepo:    orgRepo,
		memberRepo: memberRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
	}
}

// --- List ---

func TestOrgHandler_List_Success(t *testing.T) {
	env := newOrgHandlerEnv(t)

	env.orgRepo.EXPECT().ListByUser(gomock.Any(), "user-1").Return([]repository.OrgWithCounts{
		{
			Organization: models.Organization{ID: "org-1", Name: "Test Org", Slug: "test-org", CreatedAt: time.Now()},
			Role:         models.OrgRoleOwner,
			MemberCount:  2,
			BoardCount:   3,
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	data := body["data"].([]any)
	assert.Len(t, data, 1)
}

func TestOrgHandler_List_Unauthorized(t *testing.T) {
	env := newOrgHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs", nil)
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Create ---

func TestOrgHandler_Create_Unauthorized(t *testing.T) {
	env := newOrgHandlerEnv(t)

	body := `{"name":"My Org","slug":"my-org"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrgHandler_Create_InvalidSlug(t *testing.T) {
	env := newOrgHandlerEnv(t)

	body := `{"name":"My Org","slug":"a"}` // slug min=3
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Create(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// --- Update ---

func TestOrgHandler_Update_Success(t *testing.T) {
	env := newOrgHandlerEnv(t)

	// RequireOrgRole(OWNER)
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleOwner,
	}, nil)

	env.orgRepo.EXPECT().Update(gomock.Any(), "org-1", "New Name").Return(&models.Organization{
		ID: "org-1", Name: "New Name", Slug: "my-org",
	}, nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/orgs/org-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&respBody))
	data := respBody["data"].(map[string]any)
	assert.Equal(t, "New Name", data["name"])
}

func TestOrgHandler_Update_Unauthorized(t *testing.T) {
	env := newOrgHandlerEnv(t)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/orgs/org-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Update(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrgHandler_Update_Forbidden(t *testing.T) {
	env := newOrgHandlerEnv(t)

	// Not OWNER
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleMember,
	}, nil)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/orgs/org-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Update(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- Delete ---

func TestOrgHandler_Delete_Success(t *testing.T) {
	env := newOrgHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleOwner,
	}, nil)

	env.memberRepo.EXPECT().CountByUser(gomock.Any(), "user-1").Return(2, nil)
	env.orgRepo.EXPECT().BoardCount(gomock.Any(), "org-1").Return(0, nil)
	env.orgRepo.EXPECT().Delete(gomock.Any(), "org-1").Return(nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestOrgHandler_Delete_Unauthorized(t *testing.T) {
	env := newOrgHandlerEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1", nil)
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrgHandler_Delete_LastOrg(t *testing.T) {
	env := newOrgHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleOwner,
	}, nil)

	env.memberRepo.EXPECT().CountByUser(gomock.Any(), "user-1").Return(1, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	// Can't delete last org
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestOrgHandler_Delete_HasBoards(t *testing.T) {
	env := newOrgHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleOwner,
	}, nil)

	env.memberRepo.EXPECT().CountByUser(gomock.Any(), "user-1").Return(2, nil)
	env.orgRepo.EXPECT().BoardCount(gomock.Any(), "org-1").Return(3, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestOrgHandler_Delete_NoOrgID(t *testing.T) {
	env := newOrgHandlerEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOrgHandler_Delete_NotOrgOwner(t *testing.T) {
	env := newOrgHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Delete(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
