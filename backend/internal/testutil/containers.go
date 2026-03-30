package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/darwinbatres/drawgo/backend/internal/database"
)

// TestDB holds a Postgres test container and its connection pool.
type TestDB struct {
	Pool      *pgxpool.Pool
	Container *tcpg.PostgresContainer
	URL       string
}

// NewTestDB starts a Postgres container, runs migrations, and returns a pool.
// The container is automatically terminated when the test ends.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpg.Run(ctx,
		"postgres:16-alpine",
		tcpg.WithDatabase("testdb"),
		tcpg.WithUsername("test"),
		tcpg.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Run migrations
	log := zerolog.Nop()
	if err := database.Migrate(connStr, log); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	return &TestDB{Pool: pool, Container: container, URL: connStr}
}

// TruncateAll truncates all application tables to reset state between tests.
func (tdb *TestDB) TruncateAll(t *testing.T) {
	t.Helper()
	tables := []string{
		"board_assets",
		"board_permissions",
		"board_versions",
		"share_links",
		"boards",
		"memberships",
		"organizations",
		"accounts",
		"refresh_tokens",
		"audit_events",
		"backup_metadata",
		"backup_schedule",
		"users",
	}
	for _, table := range tables {
		_, err := tdb.Pool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("failed to truncate %s: %v", table, err)
		}
	}
}

// TestS3 holds a MinIO test container and its client.
type TestS3 struct {
	Client    *minio.Client
	Container *tcminio.MinioContainer
	Endpoint  string
}

// NewTestS3 starts a MinIO container and returns a client.
// The container is automatically terminated when the test ends.
func NewTestS3(t *testing.T) *TestS3 {
	t.Helper()
	ctx := context.Background()

	container, err := tcminio.Run(ctx,
		"minio/minio:RELEASE.2024-01-16T16-07-38Z",
		tcminio.WithUsername("minioadmin"),
		tcminio.WithPassword("minioadmin"),
	)
	if err != nil {
		t.Fatalf("failed to start minio container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get minio endpoint: %v", err)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("failed to create minio client: %v", err)
	}

	return &TestS3{Client: client, Container: container, Endpoint: endpoint}
}

// EnsureBucket creates a bucket if it doesn't exist.
func (ts3 *TestS3) EnsureBucket(t *testing.T, bucket string) {
	t.Helper()
	ctx := context.Background()
	exists, err := ts3.Client.BucketExists(ctx, bucket)
	if err != nil {
		t.Fatalf("failed to check bucket: %v", err)
	}
	if !exists {
		if err := ts3.Client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("failed to create bucket %s: %v", bucket, err)
		}
	}
}
