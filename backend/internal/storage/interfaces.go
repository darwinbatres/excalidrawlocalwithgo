package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

//go:generate mockgen -destination=../testutil/mocks/mock_storage.go -package=mocks github.com/darwinbatres/drawgo/backend/internal/storage ObjectStorage

// ObjectStorage defines S3-compatible object storage operations.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignedURL(ctx context.Context, key string) (string, error)
	UploadToBucket(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error
	PresignedURLFromBucket(ctx context.Context, bucket, key string) (string, error)
	DownloadFromBucket(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	DeleteFromBucket(ctx context.Context, bucket, key string) error
	ListObjects(ctx context.Context, bucket, prefix string) ([]minio.ObjectInfo, error)
	BucketStats(ctx context.Context, bucket string) (*BucketStatsResult, error)
	Ping(ctx context.Context) error
}

// BucketStatsResult contains storage metrics for a single S3 bucket.
type BucketStatsResult struct {
	Bucket       string `json:"bucket"`
	ObjectCount  int64  `json:"objectCount"`
	TotalBytes   int64  `json:"totalBytes"`
	LargestBytes int64  `json:"largestBytes"`
}
