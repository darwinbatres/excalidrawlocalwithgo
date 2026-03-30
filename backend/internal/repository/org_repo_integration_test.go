package repository_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
)

func setupOrgRepo(t *testing.T) (*repository.OrgRepository, *testutil.TestDB) {
	t.Helper()
	testutil.SkipIfNoDocker(t)
	tdb := testutil.NewTestDB(t)
	t.Cleanup(func() { tdb.TruncateAll(t) })
	return repository.NewOrgRepository(tdb.Pool), tdb
}

func TestOrgRepo_Create(t *testing.T) {
	repo, _ := setupOrgRepo(t)
	ctx := context.Background()

	org, err := repo.Create(ctx, "Acme Corp", "acme-corp")
	require.NoError(t, err)
	assert.NotEmpty(t, org.ID)
	assert.Equal(t, "Acme Corp", org.Name)
	assert.Equal(t, "acme-corp", org.Slug)
}

func TestOrgRepo_Create_DuplicateSlug(t *testing.T) {
	repo, _ := setupOrgRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, "Org1", "same-slug")
	require.NoError(t, err)

	_, err = repo.Create(ctx, "Org2", "same-slug")
	require.Error(t, err)
}

func TestOrgRepo_GetByID(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	org := testutil.OrgFixture(t, tdb.Pool, "Fetch Org", "fetch-org")
	got, err := repo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, got.ID)
	assert.Equal(t, "Fetch Org", got.Name)
	assert.Equal(t, "fetch-org", got.Slug)
}

func TestOrgRepo_GetByID_NotFound(t *testing.T) {
	repo, _ := setupOrgRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestOrgRepo_SlugExists(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	testutil.OrgFixture(t, tdb.Pool, "Slug Org", "exists-slug")

	exists, err := repo.SlugExists(ctx, "exists-slug")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.SlugExists(ctx, "no-slug")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestOrgRepo_ListByUser(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	user := testutil.UserFixture(t, tdb.Pool, "listuser@example.com")
	org1 := testutil.OrgFixture(t, tdb.Pool, "Alpha", "alpha")
	org2 := testutil.OrgFixture(t, tdb.Pool, "Beta", "beta")
	testutil.MembershipFixture(t, tdb.Pool, org1.ID, user.ID, models.OrgRoleOwner)
	testutil.MembershipFixture(t, tdb.Pool, org2.ID, user.ID, models.OrgRoleMember)

	orgs, err := repo.ListByUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, orgs, 2)
	// Ordered by name ASC
	assert.Equal(t, "Alpha", orgs[0].Name)
	assert.Equal(t, models.OrgRoleOwner, orgs[0].Role)
	assert.Equal(t, "Beta", orgs[1].Name)
	assert.Equal(t, models.OrgRoleMember, orgs[1].Role)
}

func TestOrgRepo_ListByUser_Empty(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	user := testutil.UserFixture(t, tdb.Pool, "alone@example.com")
	orgs, err := repo.ListByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, orgs)
}

func TestOrgRepo_Update(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	org := testutil.OrgFixture(t, tdb.Pool, "Old Name", "old-slug")
	updated, err := repo.Update(ctx, org.ID, "New Name")
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "old-slug", updated.Slug) // slug unchanged
}

func TestOrgRepo_Update_NotFound(t *testing.T) {
	repo, _ := setupOrgRepo(t)
	ctx := context.Background()

	_, err := repo.Update(ctx, "nonexistent", "New Name")
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestOrgRepo_Delete(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	org := testutil.OrgFixture(t, tdb.Pool, "Delete Me", "delete-me")
	err := repo.Delete(ctx, org.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, org.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestOrgRepo_BoardCount(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	user := testutil.UserFixture(t, tdb.Pool, "boards@example.com")
	org := testutil.OrgFixture(t, tdb.Pool, "Board Org", "board-org")
	testutil.BoardFixture(t, tdb.Pool, org.ID, user.ID, "Board 1")
	testutil.BoardFixture(t, tdb.Pool, org.ID, user.ID, "Board 2")

	count, err := repo.BoardCount(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestOrgRepo_BoardCount_Empty(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	org := testutil.OrgFixture(t, tdb.Pool, "Empty Org", "empty-org")
	count, err := repo.BoardCount(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestOrgRepo_CreateInTx(t *testing.T) {
	repo, tdb := setupOrgRepo(t)
	ctx := context.Background()

	tx, err := tdb.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	org, err := repo.CreateInTx(ctx, tx, "Tx Org", "tx-org")
	require.NoError(t, err)
	assert.NotEmpty(t, org.ID)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "Tx Org", got.Name)
}
