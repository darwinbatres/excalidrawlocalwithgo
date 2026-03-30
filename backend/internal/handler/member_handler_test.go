package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type memberHandlerEnv struct {
	handler    *handler.MemberHandler
	orgRepo    *mocks.MockOrgRepo
	memberRepo *mocks.MockMembershipRepo
	userRepo   *mocks.MockUserRepo
	auditRepo  *mocks.MockAuditRepo
}

func newMemberHandlerEnv(t *testing.T) *memberHandlerEnv {
	ctrl := gomock.NewController(t)
	orgRepo := mocks.NewMockOrgRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	userRepo := mocks.NewMockUserRepo(ctrl)
	auditRepo := mocks.NewMockAuditRepo(ctrl)

	accessSvc := service.NewAccessService(memberRepo)

	orgSvc := service.NewOrgService(nil, orgRepo, memberRepo, userRepo, auditRepo, accessSvc, testutil.NopLogger())
	h := handler.NewMemberHandler(orgSvc)

	return &memberHandlerEnv{
		handler:    h,
		orgRepo:    orgRepo,
		memberRepo: memberRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
	}
}

// --- List ---

func TestMemberHandler_List_Success(t *testing.T) {
	env := newMemberHandlerEnv(t)

	// RequireOrgRole(VIEWER)
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleViewer,
	}, nil)

	env.memberRepo.EXPECT().ListByOrg(gomock.Any(), "org-1").Return([]models.MembershipWithUser{
		{
			Membership: models.Membership{ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleOwner},
			User:       models.UserPublic{ID: "user-1", Email: "owner@test.com"},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/members", nil)
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

func TestMemberHandler_List_Unauthorized(t *testing.T) {
	env := newMemberHandlerEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/members", nil)
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMemberHandler_List_NoOrgAccess(t *testing.T) {
	env := newMemberHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/org-1/members", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.List(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Invite ---

func TestMemberHandler_Invite_Success(t *testing.T) {
	env := newMemberHandlerEnv(t)

	// RequireOrgRole(ADMIN)
	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	name := "Invitee"
	env.userRepo.EXPECT().GetByEmail(gomock.Any(), "invitee@test.com").Return(&models.User{
		ID: "user-2", Email: "invitee@test.com", Name: &name,
	}, nil)

	env.memberRepo.EXPECT().Exists(gomock.Any(), "org-1", "user-2").Return(false, nil)

	env.memberRepo.EXPECT().Create(gomock.Any(), "org-1", "user-2", models.OrgRoleMember).Return(&models.Membership{
		ID: "m-2", OrgID: "org-1", UserID: "user-2", Role: models.OrgRoleMember,
	}, nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	body := `{"email":"invitee@test.com","role":"MEMBER"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Invite(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var respBody map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&respBody))
	data := respBody["data"].(map[string]any)
	assert.Equal(t, "user-2", data["userId"])
}

func TestMemberHandler_Invite_Unauthorized(t *testing.T) {
	env := newMemberHandlerEnv(t)

	body := `{"email":"invitee@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Invite(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMemberHandler_Invite_NotAdmin(t *testing.T) {
	env := newMemberHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleMember, // Not admin
	}, nil)

	body := `{"email":"invitee@test.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Invite(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMemberHandler_Invite_InvalidEmail(t *testing.T) {
	env := newMemberHandlerEnv(t)

	body := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/org-1/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1"})
	rec := httptest.NewRecorder()

	env.handler.Invite(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// --- UpdateRole ---

func TestMemberHandler_UpdateRole_Success(t *testing.T) {
	env := newMemberHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	env.memberRepo.EXPECT().GetByID(gomock.Any(), "m-2").Return(&models.Membership{
		ID: "m-2", OrgID: "org-1", UserID: "user-2", Role: models.OrgRoleMember,
	}, nil)

	env.memberRepo.EXPECT().UpdateRole(gomock.Any(), "m-2", models.OrgRoleAdmin).Return(&models.Membership{
		ID: "m-2", OrgID: "org-1", UserID: "user-2", Role: models.OrgRoleAdmin,
	}, nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	body := `{"role":"ADMIN"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/orgs/org-1/members/m-2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1", "membershipId": "m-2"})
	rec := httptest.NewRecorder()

	env.handler.UpdateRole(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMemberHandler_UpdateRole_Unauthorized(t *testing.T) {
	env := newMemberHandlerEnv(t)

	body := `{"role":"ADMIN"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/orgs/org-1/members/m-2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "org-1", "membershipId": "m-2"})
	rec := httptest.NewRecorder()

	env.handler.UpdateRole(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Remove ---

func TestMemberHandler_Remove_Success(t *testing.T) {
	env := newMemberHandlerEnv(t)

	env.memberRepo.EXPECT().GetByOrgAndUser(gomock.Any(), "org-1", "user-1").Return(&models.Membership{
		ID: "m-1", OrgID: "org-1", UserID: "user-1", Role: models.OrgRoleAdmin,
	}, nil)

	env.memberRepo.EXPECT().GetByID(gomock.Any(), "m-2").Return(&models.Membership{
		ID: "m-2", OrgID: "org-1", UserID: "user-2", Role: models.OrgRoleMember,
	}, nil)

	env.memberRepo.EXPECT().Delete(gomock.Any(), "m-2").Return(nil)

	env.auditRepo.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1/members/m-2", nil)
	req = withUserID(req, "user-1")
	req = withChiParams(req, map[string]string{"id": "org-1", "membershipId": "m-2"})
	rec := httptest.NewRecorder()

	env.handler.Remove(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestMemberHandler_Remove_Unauthorized(t *testing.T) {
	env := newMemberHandlerEnv(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/org-1/members/m-2", nil)
	req = withChiParams(req, map[string]string{"id": "org-1", "membershipId": "m-2"})
	rec := httptest.NewRecorder()

	env.handler.Remove(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
