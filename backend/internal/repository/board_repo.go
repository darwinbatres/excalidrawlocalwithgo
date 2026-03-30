package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// BoardRepository handles board persistence operations.
type BoardRepository struct {
	pool *pgxpool.Pool
}

// NewBoardRepository creates a BoardRepository with the given connection pool.
func NewBoardRepository(pool *pgxpool.Pool) *BoardRepository {
	return &BoardRepository{pool: pool}
}

// BoardSearchParams defines filters for listing/searching boards.
type BoardSearchParams struct {
	OrgID      string
	Query      string
	Tags       []string
	IsArchived *bool
	Limit      int
	Offset     int
}

// BoardSearchResult is a paginated search result.
type BoardSearchResult struct {
	Boards []models.Board `json:"boards"`
	Total  int64          `json:"total"`
}

// CreateInTx inserts a new board within an existing transaction.
func (r *BoardRepository) CreateInTx(ctx context.Context, tx pgx.Tx, board *models.Board) error {
	board.ID = xid.New().String()
	board.CreatedAt = time.Now()
	board.UpdatedAt = board.CreatedAt

	_, err := tx.Exec(ctx,
		`INSERT INTO boards (id, org_id, owner_id, title, description, tags, is_archived, thumbnail, search_content, current_version_id, version_number, etag, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		board.ID, board.OrgID, board.OwnerID, board.Title, board.Description,
		board.Tags, board.IsArchived, board.Thumbnail, board.SearchContent,
		board.CurrentVersionID, board.VersionNumber, board.Etag,
		board.CreatedAt, board.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create board: %w", err)
	}
	return nil
}

// GetByID retrieves a board by its ID.
func (r *BoardRepository) GetByID(ctx context.Context, id string) (*models.Board, error) {
	return r.scanBoard(r.pool.QueryRow(ctx,
		`SELECT id, org_id, owner_id, title, description, tags, is_archived, thumbnail, search_content, current_version_id, version_number, etag, created_at, updated_at
		 FROM boards WHERE id = $1`, id,
	))
}

// GetByIDForUpdate retrieves a board with a row-level lock for optimistic concurrency.
// Must be called within a transaction.
func (r *BoardRepository) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id string) (*models.Board, error) {
	return r.scanBoard(tx.QueryRow(ctx,
		`SELECT id, org_id, owner_id, title, description, tags, is_archived, thumbnail, search_content, current_version_id, version_number, etag, created_at, updated_at
		 FROM boards WHERE id = $1 FOR UPDATE`, id,
	))
}

// GetByIDWithVersion retrieves a board joined with its latest version data.
func (r *BoardRepository) GetByIDWithVersion(ctx context.Context, id string) (*models.BoardWithVersion, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT b.id, b.org_id, b.owner_id, b.title, b.description, b.tags, b.is_archived, b.thumbnail, b.search_content, b.current_version_id, b.version_number, b.etag, b.created_at, b.updated_at,
		        v.id, v.board_id, v.version, v.created_by_id, v.label, v.created_at, v.scene_json, v.app_state_json, v.thumbnail_url
		 FROM boards b
		 LEFT JOIN board_versions v ON v.id = b.current_version_id
		 WHERE b.id = $1`, id,
	)

	var bv models.BoardWithVersion
	var vID, vBoardID, vCreatedByID *string
	var vVersion *int
	var vLabel, vThumbnailURL *string
	var vCreatedAt *time.Time
	var vSceneJSON, vAppStateJSON []byte

	err := row.Scan(
		&bv.ID, &bv.OrgID, &bv.OwnerID, &bv.Title, &bv.Description,
		&bv.Tags, &bv.IsArchived, &bv.Thumbnail, &bv.SearchContent,
		&bv.CurrentVersionID, &bv.VersionNumber, &bv.Etag,
		&bv.CreatedAt, &bv.UpdatedAt,
		&vID, &vBoardID, &vVersion, &vCreatedByID, &vLabel,
		&vCreatedAt, &vSceneJSON, &vAppStateJSON, &vThumbnailURL,
	)
	if err != nil {
		return nil, err
	}

	if vID != nil {
		bv.LatestVersion = &models.BoardVersion{
			ID:           *vID,
			BoardID:      *vBoardID,
			Version:      *vVersion,
			CreatedByID:  *vCreatedByID,
			Label:        vLabel,
			CreatedAt:    *vCreatedAt,
			SceneJSON:    vSceneJSON,
			AppStateJSON: vAppStateJSON,
			ThumbnailURL: vThumbnailURL,
		}
	}

	return &bv, nil
}

// Update updates board metadata fields.
func (r *BoardRepository) Update(ctx context.Context, board *models.Board) error {
	board.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE boards SET title = $2, description = $3, tags = $4, is_archived = $5, thumbnail = $6, updated_at = $7
		 WHERE id = $1`,
		board.ID, board.Title, board.Description, board.Tags, board.IsArchived, board.Thumbnail, board.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update board: %w", err)
	}
	return nil
}

// UpdateVersionInfoInTx updates version-related fields after a save, within a transaction.
func (r *BoardRepository) UpdateVersionInfoInTx(ctx context.Context, tx pgx.Tx, boardID, versionID, etag string, versionNumber int, searchContent *string, thumbnail *string) error {
	_, err := tx.Exec(ctx,
		`UPDATE boards SET current_version_id = $2, version_number = $3, etag = $4, search_content = $5, thumbnail = COALESCE($6, thumbnail), updated_at = $7
		 WHERE id = $1`,
		boardID, versionID, versionNumber, etag, searchContent, thumbnail, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("update board version info: %w", err)
	}
	return nil
}

// Delete permanently removes a board by ID. Relies on DB cascade for versions/permissions/assets.
func (r *BoardRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM boards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete board: %w", err)
	}
	return nil
}

// Search finds boards in an org with optional text search and tag filtering.
func (r *BoardRepository) Search(ctx context.Context, params BoardSearchParams) (*BoardSearchResult, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
	args = append(args, params.OrgID)
	argIdx++

	if params.IsArchived != nil {
		conditions = append(conditions, fmt.Sprintf("is_archived = $%d", argIdx))
		args = append(args, *params.IsArchived)
		argIdx++
	}

	if params.Query != "" {
		searchPattern := "%" + sanitizeLikePattern(params.Query) + "%"
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d OR search_content ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	if len(params.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, params.Tags)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count total matches
	var total int64
	countQuery := "SELECT COUNT(*) FROM boards " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch page
	dataQuery := fmt.Sprintf(
		`SELECT id, org_id, owner_id, title, description, tags, is_archived, thumbnail, search_content, current_version_id, version_number, etag, created_at, updated_at
		 FROM boards %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	boards := make([]models.Board, 0)
	for rows.Next() {
		b, err := r.scanBoardFromRows(rows)
		if err != nil {
			return nil, err
		}
		boards = append(boards, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &BoardSearchResult{Boards: boards, Total: total}, nil
}

// scanBoard scans a single board row.
func (r *BoardRepository) scanBoard(row pgx.Row) (*models.Board, error) {
	var b models.Board
	err := row.Scan(
		&b.ID, &b.OrgID, &b.OwnerID, &b.Title, &b.Description,
		&b.Tags, &b.IsArchived, &b.Thumbnail, &b.SearchContent,
		&b.CurrentVersionID, &b.VersionNumber, &b.Etag,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// scanBoardFromRows scans a board from a pgx.Rows iterator.
func (r *BoardRepository) scanBoardFromRows(rows pgx.Rows) (*models.Board, error) {
	var b models.Board
	err := rows.Scan(
		&b.ID, &b.OrgID, &b.OwnerID, &b.Title, &b.Description,
		&b.Tags, &b.IsArchived, &b.Thumbnail, &b.SearchContent,
		&b.CurrentVersionID, &b.VersionNumber, &b.Etag,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// sanitizeLikePattern escapes special SQL LIKE characters to prevent injection.
func sanitizeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
