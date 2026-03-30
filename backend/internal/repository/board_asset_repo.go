package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// BoardAssetRepository handles board asset persistence operations.
type BoardAssetRepository struct {
	pool *pgxpool.Pool
}

// NewBoardAssetRepository creates a BoardAssetRepository with the given connection pool.
func NewBoardAssetRepository(pool *pgxpool.Pool) *BoardAssetRepository {
	return &BoardAssetRepository{pool: pool}
}

// Upsert inserts or updates a board asset (on conflict of board_id + file_id).
// This allows re-uploading the same file to update its content.
func (r *BoardAssetRepository) Upsert(ctx context.Context, asset *models.BoardAsset) error {
	if asset.ID == "" {
		asset.ID = xid.New().String()
	}
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = time.Now()
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO board_assets (id, board_id, file_id, mime_type, size_bytes, storage_key, sha256, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (board_id, file_id)
		 DO UPDATE SET mime_type = EXCLUDED.mime_type, size_bytes = EXCLUDED.size_bytes,
		              storage_key = EXCLUDED.storage_key, sha256 = EXCLUDED.sha256, created_at = EXCLUDED.created_at`,
		asset.ID, asset.BoardID, asset.FileID, asset.MimeType,
		asset.SizeBytes, asset.StorageKey, asset.SHA256, asset.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert board asset: %w", err)
	}
	return nil
}

// GetByBoardAndFileID retrieves a single asset by board ID and file ID.
func (r *BoardAssetRepository) GetByBoardAndFileID(ctx context.Context, boardID, fileID string) (*models.BoardAsset, error) {
	return r.scanAsset(r.pool.QueryRow(ctx,
		`SELECT id, board_id, file_id, mime_type, size_bytes, storage_key, sha256, created_at
		 FROM board_assets WHERE board_id = $1 AND file_id = $2`, boardID, fileID,
	))
}

// ListByBoard retrieves all assets for a board.
func (r *BoardAssetRepository) ListByBoard(ctx context.Context, boardID string) ([]models.BoardAsset, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, board_id, file_id, mime_type, size_bytes, storage_key, sha256, created_at
		 FROM board_assets WHERE board_id = $1 ORDER BY created_at ASC`, boardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.BoardAsset
	for rows.Next() {
		a, err := r.scanAssetFromRows(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *a)
	}
	return assets, rows.Err()
}

// Delete removes an asset by ID.
func (r *BoardAssetRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM board_assets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete board asset: %w", err)
	}
	return nil
}

// DeleteByBoardAndFileID removes an asset by board ID and file ID.
func (r *BoardAssetRepository) DeleteByBoardAndFileID(ctx context.Context, boardID, fileID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM board_assets WHERE board_id = $1 AND file_id = $2`, boardID, fileID,
	)
	if err != nil {
		return fmt.Errorf("delete board asset by file id: %w", err)
	}
	return nil
}

// StorageUsedByBoard returns the total bytes used by assets belonging to a board.
func (r *BoardAssetRepository) StorageUsedByBoard(ctx context.Context, boardID string) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(size_bytes), 0) FROM board_assets WHERE board_id = $1`, boardID,
	).Scan(&total)
	return total, err
}

// StorageUsedByOrg returns the total bytes used by assets belonging to all boards in an org.
func (r *BoardAssetRepository) StorageUsedByOrg(ctx context.Context, orgID string) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(ba.size_bytes), 0) FROM board_assets ba
		 JOIN boards b ON b.id = ba.board_id WHERE b.org_id = $1`, orgID,
	).Scan(&total)
	return total, err
}

// FindBySHA256 finds an existing asset by SHA256 hash (for dedup within a board).
func (r *BoardAssetRepository) FindBySHA256(ctx context.Context, boardID, sha256 string) (*models.BoardAsset, error) {
	return r.scanAsset(r.pool.QueryRow(ctx,
		`SELECT id, board_id, file_id, mime_type, size_bytes, storage_key, sha256, created_at
		 FROM board_assets WHERE board_id = $1 AND sha256 = $2`, boardID, sha256,
	))
}

// DeleteOrphaned removes assets not referenced in the provided file ID set.
func (r *BoardAssetRepository) DeleteOrphaned(ctx context.Context, boardID string, activeFileIDs []string) ([]models.BoardAsset, error) {
	if len(activeFileIDs) == 0 {
		// All assets are orphaned — get them then delete
		orphaned, err := r.ListByBoard(ctx, boardID)
		if err != nil {
			return nil, err
		}
		if len(orphaned) > 0 {
			_, err = r.pool.Exec(ctx, `DELETE FROM board_assets WHERE board_id = $1`, boardID)
			if err != nil {
				return nil, err
			}
		}
		return orphaned, nil
	}

	// Get orphaned assets first so we can clean up S3
	rows, err := r.pool.Query(ctx,
		`SELECT id, board_id, file_id, mime_type, size_bytes, storage_key, sha256, created_at
		 FROM board_assets WHERE board_id = $1 AND file_id != ALL($2)`, boardID, activeFileIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orphaned []models.BoardAsset
	for rows.Next() {
		a, err := r.scanAssetFromRows(rows)
		if err != nil {
			return nil, err
		}
		orphaned = append(orphaned, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(orphaned) > 0 {
		_, err = r.pool.Exec(ctx,
			`DELETE FROM board_assets WHERE board_id = $1 AND file_id != ALL($2)`, boardID, activeFileIDs,
		)
		if err != nil {
			return nil, err
		}
	}

	return orphaned, nil
}

func (r *BoardAssetRepository) scanAsset(row pgx.Row) (*models.BoardAsset, error) {
	var a models.BoardAsset
	err := row.Scan(
		&a.ID, &a.BoardID, &a.FileID, &a.MimeType,
		&a.SizeBytes, &a.StorageKey, &a.SHA256, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *BoardAssetRepository) scanAssetFromRows(rows pgx.Rows) (*models.BoardAsset, error) {
	var a models.BoardAsset
	err := rows.Scan(
		&a.ID, &a.BoardID, &a.FileID, &a.MimeType,
		&a.SizeBytes, &a.StorageKey, &a.SHA256, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
