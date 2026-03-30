package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/service"
	"github.com/darwinbatres/drawgo/backend/internal/storage"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
	"github.com/darwinbatres/drawgo/backend/internal/testutil/mocks"
)

type auditTestEnv struct {
	svc        *service.AuditService
	auditRepo  *mocks.MockAuditRepo
	assetRepo  *mocks.MockBoardAssetRepo
	backupRepo *mocks.MockBackupRepo
	s3         *mocks.MockObjectStorage
}

func newAuditTestEnv(t *testing.T) *auditTestEnv {
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
	svc := service.NewAuditService(auditRepo, assetRepo, backupRepo, accessSvc, s3, testutil.NopLogger(), "data-bucket", "backup-bucket")

	return &auditTestEnv{svc: svc, auditRepo: auditRepo, assetRepo: assetRepo, backupRepo: backupRepo, s3: s3}
}

func TestAuditService_SystemStats_Success(t *testing.T) {
	env := newAuditTestEnv(t)
	ctx := context.Background()

	env.auditRepo.EXPECT().GetSystemStats(ctx).Return(&repository.SystemStatsResult{
		Counts: repository.SystemCounts{
			Users:  5,
			Boards: 3,
		},
		Database: repository.DatabaseStats{
			DatabaseSizeBytes: 500000,
			CacheHitRatio:     0.98,
			Uptime:            "1 day",
		},
	}, nil)

	env.s3.EXPECT().BucketStats(ctx, "data-bucket").Return(&storage.BucketStatsResult{
		Bucket:       "data-bucket",
		ObjectCount:  100,
		TotalBytes:   1024000,
		LargestBytes: 50000,
	}, nil)

	env.s3.EXPECT().BucketStats(ctx, "backup-bucket").Return(&storage.BucketStatsResult{
		Bucket:       "backup-bucket",
		ObjectCount:  20,
		TotalBytes:   204800,
		LargestBytes: 30000,
	}, nil)

	result, apiErr := env.svc.SystemStats(ctx)

	require.Nil(t, apiErr)
	assert.Equal(t, int64(5), result.Counts.Users)
	assert.Equal(t, int64(3), result.Counts.Boards)

	// S3 storage aggregated
	assert.Len(t, result.Storage.Buckets, 2)
	assert.Equal(t, int64(120), result.Storage.TotalObjects)
	assert.Equal(t, int64(1228800), result.Storage.TotalBytes)

	// Process stats populated
	assert.Greater(t, result.Process.PID, 0)
	assert.NotEmpty(t, result.Process.StartTime)

	// Build info populated
	assert.NotEmpty(t, result.Build.GoVersion)
}

func TestAuditService_SystemStats_RepoError(t *testing.T) {
	env := newAuditTestEnv(t)
	ctx := context.Background()

	env.auditRepo.EXPECT().GetSystemStats(ctx).Return(nil, errors.New("db error"))

	result, apiErr := env.svc.SystemStats(ctx)

	assert.Nil(t, result)
	require.NotNil(t, apiErr)
}

func TestAuditService_SystemStats_S3BucketStatsError(t *testing.T) {
	env := newAuditTestEnv(t)
	ctx := context.Background()

	env.auditRepo.EXPECT().GetSystemStats(ctx).Return(&repository.SystemStatsResult{
		Counts: repository.SystemCounts{Users: 1},
	}, nil)

	// BucketStats fails for both — should continue gracefully
	env.s3.EXPECT().BucketStats(ctx, "data-bucket").Return(nil, errors.New("s3 error"))
	env.s3.EXPECT().BucketStats(ctx, "backup-bucket").Return(nil, errors.New("s3 error"))

	result, apiErr := env.svc.SystemStats(ctx)

	require.Nil(t, apiErr)
	assert.Equal(t, int64(1), result.Counts.Users)
	assert.Empty(t, result.Storage.Buckets)
	assert.Equal(t, int64(0), result.Storage.TotalBytes)
}

func TestAuditService_SystemStats_SameBucketDedup(t *testing.T) {
	ctrl := gomock.NewController(t)
	auditRepo := mocks.NewMockAuditRepo(ctrl)
	assetRepo := mocks.NewMockBoardAssetRepo(ctrl)
	memberRepo := mocks.NewMockMembershipRepo(ctrl)
	backupRepo := mocks.NewMockBackupRepo(ctrl)
	s3 := mocks.NewMockObjectStorage(ctrl)
	accessSvc := service.NewAccessService(memberRepo)

	// Default AnyTimes for best-effort calls
	auditRepo.EXPECT().GlobalCRUDBreakdown(gomock.Any()).Return(&repository.CRUDBreakdown{}, nil).AnyTimes()
	backupRepo.EXPECT().ListMetadata(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, 0, nil).AnyTimes()
	backupRepo.EXPECT().GetSchedule(gomock.Any()).Return(&models.BackupSchedule{}, nil).AnyTimes()

	// Same bucket for both data and backup
	svc := service.NewAuditService(auditRepo, assetRepo, backupRepo, accessSvc, s3, testutil.NopLogger(), "shared-bucket", "shared-bucket")
	ctx := context.Background()

	auditRepo.EXPECT().GetSystemStats(ctx).Return(&repository.SystemStatsResult{}, nil)

	// Should only call BucketStats once for shared bucket
	s3.EXPECT().BucketStats(ctx, "shared-bucket").Return(&storage.BucketStatsResult{
		Bucket:      "shared-bucket",
		ObjectCount: 50,
		TotalBytes:  500000,
	}, nil)

	result, apiErr := svc.SystemStats(ctx)

	require.Nil(t, apiErr)
	assert.Len(t, result.Storage.Buckets, 1)
	assert.Equal(t, int64(50), result.Storage.TotalObjects)
}
