package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/jwt"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

func newAuthService(t *testing.T) (*service.AuthService, *mocks.MockUserRepo, *mocks.MockRefreshTokenRepo, *mocks.MockAuditRepo) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepo(ctrl)
	tokens := mocks.NewMockRefreshTokenRepo(ctrl)
	audit := mocks.NewMockAuditRepo(ctrl)
	jwtMgr := jwt.NewManager("test-secret-32-bytes-long-enough", 15*time.Minute, 720*time.Hour)
	svc := service.NewAuthService(users, tokens, audit, jwtMgr, testutil.NopLogger())
	return svc, users, tokens, audit
}

func TestRegister_Success(t *testing.T) {
	svc, users, tokens, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().ExistsByEmail(ctx, "test@example.com").Return(false, nil)
	users.EXPECT().Create(ctx, "test@example.com", gomock.Any(), gomock.Any()).
		Return(&models.User{
			ID:    "user-1",
			Email: "test@example.com",
		}, nil)
	tokens.EXPECT().Create(ctx, "user-1", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&models.RefreshToken{ID: "rt-1"}, nil)

	user, pair, err := svc.Register(ctx, "test@example.com", "StrongPass123!", nil)

	require.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Greater(t, pair.ExpiresIn, 0)
}

func TestRegister_EmailTaken(t *testing.T) {
	svc, users, _, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().ExistsByEmail(ctx, "taken@example.com").Return(true, nil)

	_, _, err := svc.Register(ctx, "taken@example.com", "password", nil)

	assert.ErrorIs(t, err, apierror.ErrEmailTaken)
}

func TestRegister_DBError(t *testing.T) {
	svc, users, _, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().ExistsByEmail(ctx, "test@example.com").Return(false, errors.New("db down"))

	_, _, err := svc.Register(ctx, "test@example.com", "password", nil)

	assert.ErrorIs(t, err, apierror.ErrInternal)
}

func TestLogin_Success(t *testing.T) {
	svc, users, tokens, audit := newAuthService(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	hashStr := string(hash)

	users.EXPECT().GetByEmail(ctx, "user@example.com").Return(&models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: &hashStr,
	}, nil)
	tokens.EXPECT().Create(ctx, "user-1", gomock.Any(), "test-agent", "127.0.0.1", gomock.Any()).
		Return(&models.RefreshToken{ID: "rt-1"}, nil)
	audit.EXPECT().Log(gomock.Any(), "", gomock.Any(), models.AuditActionAuthLogin, "user", "user-1", gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(nil).AnyTimes()

	user, pair, err := svc.Login(ctx, "user@example.com", "correct-password", "test-agent", "127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
	assert.NotEmpty(t, pair.AccessToken)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, users, _, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().GetByEmail(ctx, "nobody@example.com").Return(nil, pgx.ErrNoRows)

	_, _, err := svc.Login(ctx, "nobody@example.com", "password", "", "")

	assert.ErrorIs(t, err, apierror.ErrInvalidCredentials)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, users, _, audit := newAuthService(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	hashStr := string(hash)

	users.EXPECT().GetByEmail(ctx, "user@example.com").Return(&models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: &hashStr,
	}, nil)
	audit.EXPECT().Log(gomock.Any(), "", gomock.Any(), models.AuditActionAuthFailed, "user", "user-1", gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(nil).AnyTimes()

	_, _, err := svc.Login(ctx, "user@example.com", "wrong-password", "", "")

	assert.ErrorIs(t, err, apierror.ErrInvalidCredentials)
}

func TestLogin_OAuthOnlyUser(t *testing.T) {
	svc, users, _, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().GetByEmail(ctx, "oauth@example.com").Return(&models.User{
		ID:           "user-1",
		Email:        "oauth@example.com",
		PasswordHash: nil, // OAuth user, no password
	}, nil)

	_, _, err := svc.Login(ctx, "oauth@example.com", "password", "", "")

	assert.ErrorIs(t, err, apierror.ErrInvalidCredentials)
}

func TestRefreshTokens_Success(t *testing.T) {
	svc, users, tokens, _ := newAuthService(t)
	ctx := context.Background()

	tokens.EXPECT().IsTokenFamilyCompromised(ctx, "raw-token").Return(false, "", nil)
	tokens.EXPECT().GetByHash(ctx, "raw-token").Return(&models.RefreshToken{
		ID:     "rt-1",
		UserID: "user-1",
	}, nil)
	users.EXPECT().GetByID(ctx, "user-1").Return(&models.User{
		ID:    "user-1",
		Email: "user@example.com",
	}, nil)
	tokens.EXPECT().Create(ctx, "user-1", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&models.RefreshToken{ID: "rt-2"}, nil)
	tokens.EXPECT().Rotate(ctx, "rt-1", "rt-2").Return(nil)

	user, pair, err := svc.RefreshTokens(ctx, "raw-token", "agent", "127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
	assert.NotEmpty(t, pair.AccessToken)
}

func TestRefreshTokens_ReplayDetected(t *testing.T) {
	svc, _, tokens, _ := newAuthService(t)
	ctx := context.Background()

	tokens.EXPECT().IsTokenFamilyCompromised(ctx, "stolen-token").Return(true, "user-1", nil)
	tokens.EXPECT().RevokeAllForUser(ctx, "user-1").Return(nil)

	_, _, err := svc.RefreshTokens(ctx, "stolen-token", "", "")

	assert.ErrorIs(t, err, apierror.ErrTokenInvalid)
}

func TestRefreshTokens_InvalidToken(t *testing.T) {
	svc, _, tokens, _ := newAuthService(t)
	ctx := context.Background()

	tokens.EXPECT().IsTokenFamilyCompromised(ctx, "invalid").Return(false, "", nil)
	tokens.EXPECT().GetByHash(ctx, "invalid").Return(nil, pgx.ErrNoRows)

	_, _, err := svc.RefreshTokens(ctx, "invalid", "", "")

	assert.ErrorIs(t, err, apierror.ErrTokenInvalid)
}

func TestLogout_Success(t *testing.T) {
	svc, _, tokens, audit := newAuthService(t)
	ctx := context.Background()

	tokens.EXPECT().RevokeAllForUser(ctx, "user-1").Return(nil)
	audit.EXPECT().Log(gomock.Any(), "", gomock.Any(), models.AuditActionAuthLogout, "user", "user-1", gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(nil).AnyTimes()

	err := svc.Logout(ctx, "user-1", "127.0.0.1", "agent")

	assert.NoError(t, err)
}

func TestGetUser_Success(t *testing.T) {
	svc, users, _, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().GetByID(ctx, "user-1").Return(&models.User{
		ID:    "user-1",
		Email: "user@example.com",
	}, nil)

	user, err := svc.GetUser(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "user@example.com", user.Email)
}

func TestGetUser_NotFound(t *testing.T) {
	svc, users, _, _ := newAuthService(t)
	ctx := context.Background()

	users.EXPECT().GetByID(ctx, "missing").Return(nil, pgx.ErrNoRows)

	_, err := svc.GetUser(ctx, "missing")

	assert.ErrorIs(t, err, apierror.ErrUserNotFound)
}
