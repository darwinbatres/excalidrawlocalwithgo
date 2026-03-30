package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// BoardPermissionRepository handles board-level permission persistence.
type BoardPermissionRepository struct {
	pool *pgxpool.Pool
}

// NewBoardPermissionRepository creates a BoardPermissionRepository.
func NewBoardPermissionRepository(pool *pgxpool.Pool) *BoardPermissionRepository {
	return &BoardPermissionRepository{pool: pool}
}

// GetByBoardAndUser retrieves the board permission for a user (via their membership) on a specific board.
// Returns nil, pgx.ErrNoRows if no board-level permission exists.
func (r *BoardPermissionRepository) GetByBoardAndUser(ctx context.Context, boardID, userID string) (*models.BoardPermission, error) {
	var bp models.BoardPermission
	err := r.pool.QueryRow(ctx,
		`SELECT bp.id, bp.board_id, bp.membership_id, bp.role, bp.created_at, bp.updated_at
		 FROM board_permissions bp
		 JOIN memberships m ON m.id = bp.membership_id
		 WHERE bp.board_id = $1 AND m.user_id = $2`,
		boardID, userID,
	).Scan(&bp.ID, &bp.BoardID, &bp.MembershipID, &bp.Role, &bp.CreatedAt, &bp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &bp, nil
}
