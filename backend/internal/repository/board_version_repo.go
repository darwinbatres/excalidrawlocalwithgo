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

// BoardVersionRepository handles board version persistence operations.
type BoardVersionRepository struct {
	pool *pgxpool.Pool
}

// NewBoardVersionRepository creates a BoardVersionRepository.
func NewBoardVersionRepository(pool *pgxpool.Pool) *BoardVersionRepository {
	return &BoardVersionRepository{pool: pool}
}

// BoardVersionWithCreator includes creator user info for list responses.
type BoardVersionWithCreator struct {
	models.BoardVersionMeta
	CreatedByName  *string `json:"createdByName,omitempty"`
	CreatedByEmail string  `json:"createdByEmail"`
}

// CreateInTx inserts a new board version within an existing transaction.
func (r *BoardVersionRepository) CreateInTx(ctx context.Context, tx pgx.Tx, v *models.BoardVersion) error {
	v.ID = xid.New().String()
	v.CreatedAt = time.Now()

	_, err := tx.Exec(ctx,
		`INSERT INTO board_versions (id, board_id, version, created_by_id, label, created_at, scene_json, app_state_json, thumbnail_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		v.ID, v.BoardID, v.Version, v.CreatedByID, v.Label,
		v.CreatedAt, v.SceneJSON, v.AppStateJSON, v.ThumbnailURL,
	)
	if err != nil {
		return fmt.Errorf("create board version: %w", err)
	}
	return nil
}

// GetByID retrieves a board version by its primary key.
func (r *BoardVersionRepository) GetByID(ctx context.Context, id string) (*models.BoardVersion, error) {
	var v models.BoardVersion
	err := r.pool.QueryRow(ctx,
		`SELECT id, board_id, version, created_by_id, label, created_at, scene_json, app_state_json, thumbnail_url
		 FROM board_versions WHERE id = $1`, id,
	).Scan(&v.ID, &v.BoardID, &v.Version, &v.CreatedByID, &v.Label,
		&v.CreatedAt, &v.SceneJSON, &v.AppStateJSON, &v.ThumbnailURL,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetByBoardAndVersion retrieves a specific version of a board by composite key.
func (r *BoardVersionRepository) GetByBoardAndVersion(ctx context.Context, boardID string, version int) (*models.BoardVersion, error) {
	var v models.BoardVersion
	err := r.pool.QueryRow(ctx,
		`SELECT id, board_id, version, created_by_id, label, created_at, scene_json, app_state_json, thumbnail_url
		 FROM board_versions WHERE board_id = $1 AND version = $2`,
		boardID, version,
	).Scan(&v.ID, &v.BoardID, &v.Version, &v.CreatedByID, &v.Label,
		&v.CreatedAt, &v.SceneJSON, &v.AppStateJSON, &v.ThumbnailURL,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListByBoard returns paginated version history for a board with creator info.
func (r *BoardVersionRepository) ListByBoard(ctx context.Context, boardID string, limit, offset int) ([]BoardVersionWithCreator, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM board_versions WHERE board_id = $1`, boardID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT v.id, v.board_id, v.version, v.created_by_id, v.label, v.created_at,
		        u.name, u.email
		 FROM board_versions v
		 JOIN users u ON u.id = v.created_by_id
		 WHERE v.board_id = $1
		 ORDER BY v.version DESC
		 LIMIT $2 OFFSET $3`,
		boardID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	versions := make([]BoardVersionWithCreator, 0)
	for rows.Next() {
		var vc BoardVersionWithCreator
		if err := rows.Scan(
			&vc.ID, &vc.BoardID, &vc.Version, &vc.CreatedByID, &vc.Label, &vc.CreatedAt,
			&vc.CreatedByName, &vc.CreatedByEmail,
		); err != nil {
			return nil, 0, err
		}
		versions = append(versions, vc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

// BoardDataStorageInfo holds database-level storage breakdown for a board.
type BoardDataStorageInfo struct {
	SceneBytes     int64 `json:"sceneBytes"`
	AppStateBytes  int64 `json:"appStateBytes"`
	ThumbnailBytes int64 `json:"thumbnailBytes"`
	VersionCount   int64 `json:"versionCount"`
}

// DataStorageByBoard calculates the database storage used by all versions of a board.
func (r *BoardVersionRepository) DataStorageByBoard(ctx context.Context, boardID string) (*BoardDataStorageInfo, error) {
	var info BoardDataStorageInfo
	err := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(pg_column_size(scene_json)), 0),
			COALESCE(SUM(pg_column_size(app_state_json)), 0),
			COALESCE(SUM(pg_column_size(thumbnail_url)), 0),
			COUNT(*)
		 FROM board_versions WHERE board_id = $1`, boardID,
	).Scan(&info.SceneBytes, &info.AppStateBytes, &info.ThumbnailBytes, &info.VersionCount)
	if err != nil {
		return nil, fmt.Errorf("data storage by board: %w", err)
	}
	return &info, nil
}

// DataStorageByOrg calculates the database storage used by all boards in an org.
func (r *BoardVersionRepository) DataStorageByOrg(ctx context.Context, orgID string) (*BoardDataStorageInfo, error) {
	var info BoardDataStorageInfo
	err := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(pg_column_size(bv.scene_json)), 0),
			COALESCE(SUM(pg_column_size(bv.app_state_json)), 0),
			COALESCE(SUM(pg_column_size(bv.thumbnail_url)), 0),
			COUNT(*)
		 FROM board_versions bv
		 JOIN boards b ON b.id = bv.board_id
		 WHERE b.org_id = $1`, orgID,
	).Scan(&info.SceneBytes, &info.AppStateBytes, &info.ThumbnailBytes, &info.VersionCount)
	if err != nil {
		return nil, fmt.Errorf("data storage by org: %w", err)
	}
	return &info, nil
}

// PruneOldVersions deletes the oldest unlabeled versions for a board, keeping
// at most maxKeep total versions. Labeled versions (milestones) are preserved.
// Returns the number of versions deleted.
func (r *BoardVersionRepository) PruneOldVersions(ctx context.Context, boardID string, maxKeep int) (int64, error) {
	if maxKeep <= 0 {
		return 0, nil // 0 = unlimited
	}

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM board_versions
		 WHERE id IN (
		   SELECT id FROM board_versions
		   WHERE board_id = $1
		     AND (label IS NULL OR label = '')
		   ORDER BY version DESC
		   OFFSET $2
		 )`, boardID, maxKeep,
	)
	if err != nil {
		return 0, fmt.Errorf("prune old versions: %w", err)
	}
	return tag.RowsAffected(), nil
}
