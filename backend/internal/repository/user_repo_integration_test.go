package repository_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
)

func setupUserRepo(t *testing.T) (*repository.UserRepository, *testutil.TestDB) {
	t.Helper()
	testutil.SkipIfNoDocker(t)
	tdb := testutil.NewTestDB(t)
	t.Cleanup(func() { tdb.TruncateAll(t) })
	return repository.NewUserRepository(tdb.Pool), tdb
}

func TestUserRepo_Create(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	name := "Alice"
	hash := "$2a$12$LJ3m4ys3Lk0TSwHilbpVwuGh8jGn/cGzkVYJlIkTiYQKPmFqVOxMi"
	user, err := repo.Create(ctx, "alice@example.com", &name, &hash)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, &name, user.Name)
	assert.Nil(t, user.EmailVerified)
}

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, "dup@example.com", nil, nil)
	require.NoError(t, err)

	_, err = repo.Create(ctx, "dup@example.com", nil, nil)
	require.Error(t, err)
}

func TestUserRepo_GetByID(t *testing.T) {
	repo, tdb := setupUserRepo(t)
	ctx := context.Background()

	user := testutil.UserFixture(t, tdb.Pool, "getbyid@example.com")
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, user.Email, got.Email)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestUserRepo_GetByEmail(t *testing.T) {
	repo, tdb := setupUserRepo(t)
	ctx := context.Background()

	user := testutil.UserFixture(t, tdb.Pool, "byemail@example.com")
	got, err := repo.GetByEmail(ctx, "byemail@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nobody@example.com")
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestUserRepo_ExistsByEmail(t *testing.T) {
	repo, tdb := setupUserRepo(t)
	ctx := context.Background()

	testutil.UserFixture(t, tdb.Pool, "exists@example.com")

	exists, err := repo.ExistsByEmail(ctx, "exists@example.com")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByEmail(ctx, "nope@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserRepo_UpdateProfile(t *testing.T) {
	repo, tdb := setupUserRepo(t)
	ctx := context.Background()

	user := testutil.UserFixture(t, tdb.Pool, "update@example.com")
	newName := "Updated Name"
	newImg := "https://example.com/avatar.png"

	err := repo.UpdateProfile(ctx, user.ID, &newName, &newImg)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, &newName, got.Name)
	assert.Equal(t, &newImg, got.Image)
}

func TestUserRepo_VerifyEmail(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	// Create unverified user
	user, err := repo.Create(ctx, "verify@example.com", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, user.EmailVerified)

	err = repo.VerifyEmail(ctx, user.ID)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.EmailVerified)
}

func TestUserRepo_CreateOrGetByOAuth_New(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	name := "OAuth User"
	img := "https://example.com/photo.png"
	user, created, err := repo.CreateOrGetByOAuth(ctx, "oauth@example.com", &name, &img, true)
	require.NoError(t, err)
	assert.True(t, created)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "oauth@example.com", user.Email)
	assert.NotNil(t, user.EmailVerified)
}

func TestUserRepo_CreateOrGetByOAuth_Existing(t *testing.T) {
	repo, tdb := setupUserRepo(t)
	ctx := context.Background()

	existing := testutil.UserFixture(t, tdb.Pool, "existing@example.com")
	got, created, err := repo.CreateOrGetByOAuth(ctx, "existing@example.com", nil, nil, false)
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, existing.ID, got.ID)
}
