package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/token"
)

// ShareLinkRepository handles share link persistence operations.
type ShareLinkRepository struct {
	pool *pgxpool.Pool
}

// NewShareLinkRepository creates a ShareLinkRepository.
func NewShareLinkRepository(pool *pgxpool.Pool) *ShareLinkRepository {
	return &ShareLinkRepository{pool: pool}
}

// Create inserts a new share link with a cryptographically random token.
func (r *ShareLinkRepository) Create(ctx context.Context, boardID, createdBy string, role models.BoardRole, expiresAt *time.Time) (*models.ShareLink, error) {
	tok, err := token.Generate(32)
	if err != nil {
		return nil, err
	}

	link := &models.ShareLink{
		ID:        xid.New().String(),
		BoardID:   boardID,
		Token:     tok,
		Role:      role,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO share_links (id, board_id, token, role, expires_at, created_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		link.ID, link.BoardID, link.Token, link.Role, link.ExpiresAt, link.CreatedAt, link.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return link, nil
}

// GetByToken retrieves a share link by its unique token.
func (r *ShareLinkRepository) GetByToken(ctx context.Context, tok string) (*models.ShareLink, error) {
	var s models.ShareLink
	err := r.pool.QueryRow(ctx,
		`SELECT id, board_id, token, role, expires_at, created_at, created_by
		 FROM share_links WHERE token = $1`, tok,
	).Scan(&s.ID, &s.BoardID, &s.Token, &s.Role, &s.ExpiresAt, &s.CreatedAt, &s.CreatedBy)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// GetByID retrieves a share link by ID.
func (r *ShareLinkRepository) GetByID(ctx context.Context, id string) (*models.ShareLink, error) {
	var s models.ShareLink
	err := r.pool.QueryRow(ctx,
		`SELECT id, board_id, token, role, expires_at, created_at, created_by
		 FROM share_links WHERE id = $1`, id,
	).Scan(&s.ID, &s.BoardID, &s.Token, &s.Role, &s.ExpiresAt, &s.CreatedAt, &s.CreatedBy)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// ListByBoard returns all share links for a board, ordered newest first.
func (r *ShareLinkRepository) ListByBoard(ctx context.Context, boardID string) ([]models.ShareLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, board_id, token, role, expires_at, created_at, created_by
		 FROM share_links WHERE board_id = $1 ORDER BY created_at DESC`, boardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.ShareLink
	for rows.Next() {
		var s models.ShareLink
		if err := rows.Scan(&s.ID, &s.BoardID, &s.Token, &s.Role, &s.ExpiresAt, &s.CreatedAt, &s.CreatedBy); err != nil {
			return nil, err
		}
		links = append(links, s)
	}
	return links, rows.Err()
}

// Delete removes a share link by ID.
func (r *ShareLinkRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM share_links WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete share link: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired share links. Returns the number deleted.
func (r *ShareLinkRepository) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM share_links WHERE expires_at IS NOT NULL AND expires_at < $1`,
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteByBoard removes all share links for a board.
func (r *ShareLinkRepository) DeleteByBoard(ctx context.Context, boardID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM share_links WHERE board_id = $1`, boardID)
	if err != nil {
		return fmt.Errorf("delete share links by board: %w", err)
	}
	return nil
}
