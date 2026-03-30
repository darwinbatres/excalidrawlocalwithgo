package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type shareTestEnv struct {
	svc    *service.ShareService
	shares *mocks.MockShareLinkRepo
	boards *mocks.MockBoardRepo
	audit  *mocks.MockAuditRepo
	memb   *mocks.MockMembershipRepo
	bperm  *mocks.MockBoardPermissionRepo
}

func newShareTestEnv(t *testing.T) *shareTestEnv {
	ctrl := gomock.NewController(t)
	shares := mocks.NewMockShareLinkRepo(ctrl)
	boards := mocks.NewMockBoardRepo(ctrl)
	audit := mocks.NewMockAuditRepo(ctrl)
	memb := mocks.NewMockMembershipRepo(ctrl)
	bperm := mocks.NewMockBoardPermissionRepo(ctrl)

	access := service.NewAccessService(memb)
	access.WithBoardRepos(boards, bperm)

	svc := service.NewShareService(shares, boards, audit, access, testutil.NopLogger())
	return &shareTestEnv{svc: svc, shares: shares, boards: boards, audit: audit, memb: memb, bperm: bperm}
}

func TestCreateShareLink_Success(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	board := &models.Board{ID: "board-1", OrgID: "org-1", OwnerID: "user-1"}
	env.boards.EXPECT().GetByID(ctx, "board-1").Return(board, nil)
	env.shares.EXPECT().Create(ctx, "board-1", "user-1", models.BoardRoleViewer, gomock.Any()).
		Return(&models.ShareLink{
			ID:      "link-1",
			BoardID: "board-1",
			Token:   "abc123",
			Role:    models.BoardRoleViewer,
		}, nil)
	env.audit.EXPECT().Log(ctx, "org-1", gomock.Any(), models.AuditActionShareCreate, "share_link", "link-1", gomock.Nil(), gomock.Nil(), gomock.Any()).
		Return(nil)

	link, apiErr := env.svc.CreateShareLink(ctx, service.CreateShareLinkInput{
		BoardID: "board-1",
		UserID:  "user-1",
		Role:    models.BoardRoleViewer,
	})

	require.Nil(t, apiErr)
	assert.Equal(t, "link-1", link.ID)
	assert.Equal(t, models.BoardRoleViewer, link.Role)
}

func TestCreateShareLink_InvalidRole(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	board := &models.Board{ID: "board-1", OrgID: "org-1", OwnerID: "user-1"}
	env.boards.EXPECT().GetByID(ctx, "board-1").Return(board, nil)

	_, apiErr := env.svc.CreateShareLink(ctx, service.CreateShareLinkInput{
		BoardID: "board-1",
		UserID:  "user-1",
		Role:    models.BoardRoleOwner, // OWNER not allowed for share links
	})

	require.NotNil(t, apiErr)
}

func TestCreateShareLink_NotEditor(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	board := &models.Board{ID: "board-1", OrgID: "org-1", OwnerID: "other-user"}
	env.boards.EXPECT().GetByID(ctx, "board-1").Return(board, nil)
	// User is only a VIEWER in org, not ADMIN+
	env.memb.EXPECT().GetByOrgAndUser(ctx, "org-1", "user-2").
		Return(&models.Membership{Role: models.OrgRoleViewer}, nil)
	// No board-level permission
	env.bperm.EXPECT().GetByBoardAndUser(ctx, "board-1", "user-2").Return(nil, errors.New("not found"))

	_, apiErr := env.svc.CreateShareLink(ctx, service.CreateShareLinkInput{
		BoardID: "board-1",
		UserID:  "user-2",
		Role:    models.BoardRoleViewer,
	})

	require.NotNil(t, apiErr)
}

func TestListShareLinks_Success(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	board := &models.Board{ID: "board-1", OrgID: "org-1", OwnerID: "user-1"}
	env.boards.EXPECT().GetByID(ctx, "board-1").Return(board, nil)
	env.shares.EXPECT().ListByBoard(ctx, "board-1").Return([]models.ShareLink{
		{ID: "link-1", BoardID: "board-1"},
		{ID: "link-2", BoardID: "board-1"},
	}, nil)

	links, apiErr := env.svc.ListShareLinks(ctx, "user-1", "board-1")

	require.Nil(t, apiErr)
	assert.Len(t, links, 2)
}

func TestRevokeShareLink_Success(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	env.shares.EXPECT().GetByID(ctx, "link-1").Return(&models.ShareLink{
		ID:      "link-1",
		BoardID: "board-1",
	}, nil)
	board := &models.Board{ID: "board-1", OrgID: "org-1", OwnerID: "user-1"}
	env.boards.EXPECT().GetByID(ctx, "board-1").Return(board, nil)
	env.shares.EXPECT().Delete(ctx, "link-1").Return(nil)
	env.audit.EXPECT().Log(ctx, "org-1", gomock.Any(), models.AuditActionShareRevoke, "share_link", "link-1", gomock.Nil(), gomock.Nil(), gomock.Any()).
		Return(nil)

	apiErr := env.svc.RevokeShareLink(ctx, "user-1", "link-1")

	assert.Nil(t, apiErr)
}

func TestRevokeShareLink_NotFound(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	env.shares.EXPECT().GetByID(ctx, "link-missing").Return(nil, nil)

	apiErr := env.svc.RevokeShareLink(ctx, "user-1", "link-missing")

	require.NotNil(t, apiErr)
}

func TestGetSharedBoard_Success(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	env.shares.EXPECT().GetByToken(ctx, "valid-token").Return(&models.ShareLink{
		ID:      "link-1",
		BoardID: "board-1",
		Role:    models.BoardRoleViewer,
	}, nil)
	env.boards.EXPECT().GetByIDWithVersion(ctx, "board-1").Return(&models.BoardWithVersion{
		Board: models.Board{ID: "board-1", OrgID: "org-1"},
	}, nil)
	env.audit.EXPECT().Log(ctx, "org-1", gomock.Nil(), models.AuditActionShareAccess, "board", "board-1", gomock.Nil(), gomock.Nil(), gomock.Any()).
		Return(nil)

	data, apiErr := env.svc.GetSharedBoard(ctx, "valid-token")

	require.Nil(t, apiErr)
	assert.Equal(t, "board-1", data.Board.ID)
	assert.Equal(t, models.BoardRoleViewer, data.ShareRole)
}

func TestGetSharedBoard_ExpiredToken(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	env.shares.EXPECT().GetByToken(ctx, "expired-token").Return(&models.ShareLink{
		ID:        "link-1",
		BoardID:   "board-1",
		ExpiresAt: &expired,
	}, nil)

	_, apiErr := env.svc.GetSharedBoard(ctx, "expired-token")

	require.NotNil(t, apiErr)
}

func TestGetSharedBoard_InvalidToken(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	env.shares.EXPECT().GetByToken(ctx, "bad-token").Return(nil, nil)

	_, apiErr := env.svc.GetSharedBoard(ctx, "bad-token")

	require.NotNil(t, apiErr)
}

func TestValidateShareToken_Valid(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	env.shares.EXPECT().GetByToken(ctx, "valid").Return(&models.ShareLink{
		ID:      "link-1",
		BoardID: "board-1",
	}, nil)

	link, ok := env.svc.ValidateShareToken(ctx, "valid")

	assert.True(t, ok)
	assert.Equal(t, "link-1", link.ID)
}

func TestValidateShareToken_Invalid(t *testing.T) {
	env := newShareTestEnv(t)
	ctx := context.Background()

	env.shares.EXPECT().GetByToken(ctx, "invalid").Return(nil, errors.New("not found"))

	_, ok := env.svc.ValidateShareToken(ctx, "invalid")

	assert.False(t, ok)
}
