package service_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

func newAccessService(t *testing.T) (*service.AccessService, *mocks.MockMembershipRepo, *mocks.MockBoardRepo, *mocks.MockBoardPermissionRepo) {
	ctrl := gomock.NewController(t)
	memb := mocks.NewMockMembershipRepo(ctrl)
	boards := mocks.NewMockBoardRepo(ctrl)
	bperm := mocks.NewMockBoardPermissionRepo(ctrl)
	svc := service.NewAccessService(memb)
	svc.WithBoardRepos(boards, bperm)
	return svc, memb, boards, bperm
}

func TestRequireOrgRole_Owner(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(&models.Membership{
		ID:     "m-1",
		OrgID:  "org-1",
		UserID: "user-1",
		Role:   models.OrgRoleOwner,
	}, nil)

	m, apiErr := svc.RequireOrgRole(ctx, "user-1", "org-1", models.OrgRoleMember)
	require.Nil(t, apiErr)
	assert.Equal(t, models.OrgRoleOwner, m.Role)
}

func TestRequireOrgRole_InsufficientRole(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleViewer,
	}, nil)

	_, apiErr := svc.RequireOrgRole(ctx, "user-1", "org-1", models.OrgRoleAdmin)
	require.NotNil(t, apiErr)
}

func TestRequireOrgRole_NotMember(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(nil, pgx.ErrNoRows)

	_, apiErr := svc.RequireOrgRole(ctx, "user-1", "org-1", models.OrgRoleMember)
	require.NotNil(t, apiErr)
}

func TestHasOrgRole_True(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleAdmin,
	}, nil)

	assert.True(t, svc.HasOrgRole(ctx, "user-1", "org-1", models.OrgRoleMember))
}

func TestHasOrgRole_False(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleViewer,
	}, nil)

	assert.False(t, svc.HasOrgRole(ctx, "user-1", "org-1", models.OrgRoleAdmin))
}

func TestRequireBoardView_OwnerAccess(t *testing.T) {
	svc, _, boards, _ := newAccessService(t)
	ctx := context.Background()

	boards.EXPECT().GetByID(ctx, "board-1").Return(&models.Board{
		ID:      "board-1",
		OwnerID: "user-1",
		OrgID:   "org-1",
	}, nil)

	board, apiErr := svc.RequireBoardView(ctx, "user-1", "board-1")
	require.Nil(t, apiErr)
	assert.Equal(t, "board-1", board.ID)
}

func TestRequireBoardView_OrgMember(t *testing.T) {
	svc, memb, boards, _ := newAccessService(t)
	ctx := context.Background()

	boards.EXPECT().GetByID(ctx, "board-1").Return(&models.Board{
		ID:      "board-1",
		OwnerID: "other-user",
		OrgID:   "org-1",
	}, nil)
	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-2").Return(&models.Membership{
		Role: models.OrgRoleViewer,
	}, nil)

	board, apiErr := svc.RequireBoardView(ctx, "user-2", "board-1")
	require.Nil(t, apiErr)
	assert.Equal(t, "board-1", board.ID)
}

func TestRequireBoardView_NotFound(t *testing.T) {
	svc, _, boards, _ := newAccessService(t)
	ctx := context.Background()

	boards.EXPECT().GetByID(ctx, "missing").Return(nil, pgx.ErrNoRows)

	_, apiErr := svc.RequireBoardView(ctx, "user-1", "missing")
	require.NotNil(t, apiErr)
}

func TestRequireBoardEdit_OwnerAccess(t *testing.T) {
	svc, _, boards, _ := newAccessService(t)
	ctx := context.Background()

	boards.EXPECT().GetByID(ctx, "board-1").Return(&models.Board{
		ID:      "board-1",
		OwnerID: "user-1",
		OrgID:   "org-1",
	}, nil)

	board, apiErr := svc.RequireBoardEdit(ctx, "user-1", "board-1")
	require.Nil(t, apiErr)
	assert.Equal(t, "board-1", board.ID)
}

func TestIsOrgOwner_True(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleOwner,
	}, nil)

	assert.True(t, svc.IsOrgOwner(ctx, "user-1", "org-1"))
}

func TestIsOrgOwner_False(t *testing.T) {
	svc, memb, _, _ := newAccessService(t)
	ctx := context.Background()

	memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-1").Return(&models.Membership{
		Role: models.OrgRoleAdmin,
	}, nil)

	assert.False(t, svc.IsOrgOwner(ctx, "user-1", "org-1"))
}
