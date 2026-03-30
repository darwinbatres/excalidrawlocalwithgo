package storage_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/darwinbatres/drawgo/backend/internal/config"
	"github.com/darwinbatres/drawgo/backend/internal/storage"
	"github.com/darwinbatres/drawgo/backend/internal/testutil"
)

func setupS3(t *testing.T) *storage.S3Client {
	t.Helper()
	testutil.SkipIfNoDocker(t)
	ts3 := testutil.NewTestS3(t)

	cfg := &config.Config{
		S3Endpoint:      ts3.Endpoint,
		S3AccessKey:     "minioadmin",
		S3SecretKey:     "minioadmin",
		S3Bucket:        "test-bucket",
		S3UseSSL:        false,
		S3Region:        "us-east-1",
		S3PresignExpiry: 1 * time.Hour,
		BackupS3Bucket:  "test-backups",
	}

	log := testutil.NopLogger()
	client, err := storage.NewS3Client(context.Background(), cfg, log)
	require.NoError(t, err)
	return client
}

func TestS3_UploadAndDownload(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	data := []byte("hello, world!")
	err := s3.Upload(ctx, "test/hello.txt", bytes.NewReader(data), int64(len(data)), "text/plain")
	require.NoError(t, err)

	reader, err := s3.Download(ctx, "test/hello.txt")
	require.NoError(t, err)
	defer reader.Close()

	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestS3_Delete(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	data := []byte("to delete")
	err := s3.Upload(ctx, "test/delete-me.txt", bytes.NewReader(data), int64(len(data)), "text/plain")
	require.NoError(t, err)

	err = s3.Delete(ctx, "test/delete-me.txt")
	require.NoError(t, err)

	// Download after delete should fail (stat will error)
	reader, err := s3.Download(ctx, "test/delete-me.txt")
	if err == nil {
		// MinIO returns a reader that errors on read/stat
		_, readErr := io.ReadAll(reader)
		reader.Close()
		assert.Error(t, readErr)
	}
}

func TestS3_PresignedURL(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	data := []byte("presigned content")
	err := s3.Upload(ctx, "test/presigned.txt", bytes.NewReader(data), int64(len(data)), "text/plain")
	require.NoError(t, err)

	presigned, err := s3.PresignedURL(ctx, "test/presigned.txt")
	require.NoError(t, err)
	assert.Contains(t, presigned, "test/presigned.txt")
	assert.Contains(t, presigned, "X-Amz-Signature")
}

func TestS3_UploadToBucket(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	data := []byte("backup data")
	err := s3.UploadToBucket(ctx, "test-backups", "backups/2024/backup.sql", bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	require.NoError(t, err)

	reader, err := s3.DownloadFromBucket(ctx, "test-backups", "backups/2024/backup.sql")
	require.NoError(t, err)
	defer reader.Close()

	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestS3_DeleteFromBucket(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	data := []byte("to delete from bucket")
	err := s3.UploadToBucket(ctx, "test-backups", "del/item.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	require.NoError(t, err)

	err = s3.DeleteFromBucket(ctx, "test-backups", "del/item.bin")
	require.NoError(t, err)
}

func TestS3_PresignedURLFromBucket(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	data := []byte("bucket presigned")
	err := s3.UploadToBucket(ctx, "test-backups", "presign/file.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	require.NoError(t, err)

	presigned, err := s3.PresignedURLFromBucket(ctx, "test-backups", "presign/file.bin")
	require.NoError(t, err)
	assert.Contains(t, presigned, "presign/file.bin")
}

func TestS3_ListObjects(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	// Upload multiple objects
	for _, key := range []string{"list/a.txt", "list/b.txt", "list/c.txt"} {
		data := []byte(key)
		err := s3.UploadToBucket(ctx, "test-backups", key, bytes.NewReader(data), int64(len(data)), "text/plain")
		require.NoError(t, err)
	}

	objects, err := s3.ListObjects(ctx, "test-backups", "list/")
	require.NoError(t, err)
	assert.Len(t, objects, 3)

	keys := make([]string, len(objects))
	for i, o := range objects {
		keys[i] = o.Key
	}
	assert.Contains(t, keys, "list/a.txt")
	assert.Contains(t, keys, "list/b.txt")
	assert.Contains(t, keys, "list/c.txt")
}

func TestS3_ListObjects_Empty(t *testing.T) {
	s3 := setupS3(t)
	ctx := context.Background()

	objects, err := s3.ListObjects(ctx, "test-backups", "nonexistent/")
	require.NoError(t, err)
	assert.Empty(t, objects)
}
