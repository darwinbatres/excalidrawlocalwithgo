package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/config"
)

// S3Client wraps the MinIO SDK for S3-compatible object storage.
type S3Client struct {
	client        *minio.Client
	bucket        string
	presignExpiry time.Duration
	log           zerolog.Logger
}

// NewS3Client creates and initializes an S3-compatible client.
func NewS3Client(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*S3Client, error) {
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("creating S3 client: %w", err)
	}

	s3 := &S3Client{
		client:        client,
		bucket:        cfg.S3Bucket,
		presignExpiry: cfg.S3PresignExpiry,
		log:           log,
	}

	// Ensure bucket exists
	if err = s3.ensureBucket(ctx, cfg.S3Bucket); err != nil {
		return nil, err
	}

	// Also ensure backup bucket if configured
	if cfg.BackupS3Bucket != "" && cfg.BackupS3Bucket != cfg.S3Bucket {
		if err = s3.ensureBucket(ctx, cfg.BackupS3Bucket); err != nil {
			return nil, err
		}
	}

	log.Info().Str("endpoint", cfg.S3Endpoint).Str("bucket", cfg.S3Bucket).Msg("S3 storage initialized")
	return s3, nil
}

// Upload stores an object in the bucket.
func (s *S3Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("uploading object %s: %w", key, err)
	}
	return nil
}

// Download retrieves an object from the bucket.
func (s *S3Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("downloading object %s: %w", key, err)
	}
	return obj, nil
}

// Delete removes an object from the bucket.
func (s *S3Client) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// PresignedURL generates a temporary download URL.
func (s *S3Client) PresignedURL(ctx context.Context, key string) (string, error) {
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, s.presignExpiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %s: %w", key, err)
	}
	return u.String(), nil
}

// UploadToBucket stores an object in a specific bucket (used for backups).
func (s *S3Client) UploadToBucket(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("uploading object %s to bucket %s: %w", key, bucket, err)
	}
	return nil
}

// PresignedURLFromBucket generates a presigned URL from a specific bucket.
func (s *S3Client) PresignedURLFromBucket(ctx context.Context, bucket, key string) (string, error) {
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, bucket, key, s.presignExpiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %s/%s: %w", bucket, key, err)
	}
	return u.String(), nil
}

// DownloadFromBucket retrieves an object from a specific bucket.
func (s *S3Client) DownloadFromBucket(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("downloading object %s from bucket %s: %w", key, bucket, err)
	}
	return obj, nil
}

// DeleteFromBucket removes an object from a specific bucket.
func (s *S3Client) DeleteFromBucket(ctx context.Context, bucket, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// ListObjects returns all object keys in a bucket with a given prefix.
func (s *S3Client) ListObjects(ctx context.Context, bucket, prefix string) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo
	for obj := range s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("listing objects in %s/%s: %w", bucket, prefix, obj.Err)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// Ping checks connectivity to S3 by verifying the bucket exists.
func (s *S3Client) Ping(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucket)
	return err
}

// BucketStats returns storage metrics for a bucket by listing all objects.
func (s *S3Client) BucketStats(ctx context.Context, bucket string) (*BucketStatsResult, error) {
	result := &BucketStatsResult{Bucket: bucket}
	for obj := range s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("listing objects in %s for stats: %w", bucket, obj.Err)
		}
		result.ObjectCount++
		result.TotalBytes += obj.Size
		if obj.Size > result.LargestBytes {
			result.LargestBytes = obj.Size
		}
	}
	return result, nil
}

func (s *S3Client) ensureBucket(ctx context.Context, name string) error {
	exists, err := s.client.BucketExists(ctx, name)
	if err != nil {
		return fmt.Errorf("checking bucket %s: %w", name, err)
	}
	if !exists {
		if err = s.client.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("creating bucket %s: %w", name, err)
		}
		s.log.Info().Str("bucket", name).Msg("created S3 bucket")
	}
	return nil
}
