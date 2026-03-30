package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/config"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type backupTestEnv struct {
	svc   *service.BackupService
	repo  *mocks.MockBackupRepo
	audit *mocks.MockAuditRepo
	s3    *mocks.MockObjectStorage
}

func newBackupTestEnv(t *testing.T) *backupTestEnv {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockBackupRepo(ctrl)
	audit := mocks.NewMockAuditRepo(ctrl)
	s3 := mocks.NewMockObjectStorage(ctrl)
	cfg := &config.Config{
		BackupS3Bucket: "test-backups",
		DatabaseURL:    "postgres://test:test@localhost/testdb",
	}
	svc := service.NewBackupService(repo, audit, s3, cfg, testutil.NopLogger())
	return &backupTestEnv{svc: svc, repo: repo, audit: audit, s3: s3}
}

func TestListBackups_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().ListMetadata(ctx, 10, 0).Return([]models.BackupMetadata{
		{ID: "b-1", Status: "completed"},
		{ID: "b-2", Status: "in_progress"},
	}, 2, nil)

	backups, total, apiErr := env.svc.ListBackups(ctx, 10, 0)

	require.Nil(t, apiErr)
	assert.Len(t, backups, 2)
	assert.Equal(t, 2, total)
}

func TestListBackups_DBError(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().ListMetadata(ctx, 10, 0).Return(nil, 0, errors.New("db error"))

	_, _, apiErr := env.svc.ListBackups(ctx, 10, 0)

	require.NotNil(t, apiErr)
}

func TestGetBackup_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetMetadata(ctx, "b-1").Return(&models.BackupMetadata{
		ID:     "b-1",
		Status: "completed",
	}, nil)

	meta, apiErr := env.svc.GetBackup(ctx, "b-1")

	require.Nil(t, apiErr)
	assert.Equal(t, "b-1", meta.ID)
}

func TestGetBackup_NotFound(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetMetadata(ctx, "missing").Return(nil, errors.New("not found"))

	_, apiErr := env.svc.GetBackup(ctx, "missing")

	require.NotNil(t, apiErr)
}

func TestGetDownloadURL_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetMetadata(ctx, "b-1").Return(&models.BackupMetadata{
		ID:         "b-1",
		Status:     "completed",
		StorageKey: "backups/2025/01/backup.sql",
	}, nil)
	env.s3.EXPECT().PresignedURLFromBucket(ctx, "test-backups", "backups/2025/01/backup.sql").
		Return("https://s3.example.com/signed-url", nil)

	url, apiErr := env.svc.GetDownloadURL(ctx, "b-1")

	require.Nil(t, apiErr)
	assert.Contains(t, url, "signed-url")
}

func TestGetDownloadURL_NotCompleted(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetMetadata(ctx, "b-1").Return(&models.BackupMetadata{
		ID:     "b-1",
		Status: "in_progress",
	}, nil)

	_, apiErr := env.svc.GetDownloadURL(ctx, "b-1")

	require.NotNil(t, apiErr)
}

func TestDeleteBackup_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetMetadata(ctx, "b-1").Return(&models.BackupMetadata{
		ID:         "b-1",
		Filename:   "backup.sql",
		StorageKey: "backups/2025/01/backup.sql",
	}, nil)
	env.s3.EXPECT().DeleteFromBucket(ctx, "test-backups", "backups/2025/01/backup.sql").Return(nil)
	env.repo.EXPECT().DeleteMetadata(ctx, "b-1").Return(nil)
	env.audit.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), models.AuditActionBackupDelete, "backup", "b-1", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	apiErr := env.svc.DeleteBackup(ctx, "b-1", "actor-1")

	assert.Nil(t, apiErr)
}

func TestDeleteBackup_NotFound(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetMetadata(ctx, "missing").Return(nil, errors.New("not found"))

	apiErr := env.svc.DeleteBackup(ctx, "missing", "actor-1")

	require.NotNil(t, apiErr)
}

func TestGetSchedule_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetSchedule(ctx).Return(&models.BackupSchedule{
		Enabled:  true,
		CronExpr: "0 3 * * *",
	}, nil)

	sched, apiErr := env.svc.GetSchedule(ctx)

	require.Nil(t, apiErr)
	assert.True(t, sched.Enabled)
	assert.Equal(t, "0 3 * * *", sched.CronExpr)
}

func TestUpdateSchedule_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().UpdateSchedule(ctx, true, "0 3 * * *", 7, 4, 6).
		Return(&models.BackupSchedule{
			Enabled:     true,
			CronExpr:    "0 3 * * *",
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
		}, nil)
	env.audit.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), models.AuditActionBackupScheduleUpdate, "backup_schedule", "default", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	sched, apiErr := env.svc.UpdateSchedule(ctx, "actor-1", true, "0 3 * * *", 7, 4, 6)

	require.Nil(t, apiErr)
	assert.True(t, sched.Enabled)
	assert.Equal(t, 7, sched.KeepDaily)
}

func TestUpdateSchedule_InvalidCron(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	_, apiErr := env.svc.UpdateSchedule(ctx, "actor-1", true, "invalid cron", 7, 4, 6)

	require.NotNil(t, apiErr)
}

func TestUpdateSchedule_InvalidRetention(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	_, apiErr := env.svc.UpdateSchedule(ctx, "actor-1", true, "0 3 * * *", 0, -1, 6)

	require.NotNil(t, apiErr)
}

func TestRotateBackups_Success(t *testing.T) {
	env := newBackupTestEnv(t)
	ctx := context.Background()

	env.repo.EXPECT().GetSchedule(ctx).Return(&models.BackupSchedule{
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
	}, nil)
	env.repo.EXPECT().ListExpiredForRotation(ctx, 7, 4, 6).Return([]models.BackupMetadata{
		{ID: "old-1", StorageKey: "backups/old-1.sql"},
		{ID: "old-2", StorageKey: "backups/old-2.sql"},
	}, nil)
	env.s3.EXPECT().DeleteFromBucket(ctx, "test-backups", "backups/old-1.sql").Return(nil)
	env.repo.EXPECT().DeleteMetadata(ctx, "old-1").Return(nil)
	env.s3.EXPECT().DeleteFromBucket(ctx, "test-backups", "backups/old-2.sql").Return(nil)
	env.repo.EXPECT().DeleteMetadata(ctx, "old-2").Return(nil)

	deleted, err := env.svc.RotateBackups(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, deleted)
}
