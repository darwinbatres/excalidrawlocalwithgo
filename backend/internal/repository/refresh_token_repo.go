package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// RefreshTokenRepository handles refresh token persistence.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository creates a RefreshTokenRepository.
func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

// Create stores a new refresh token (hashed) for the given user.
func (r *RefreshTokenRepository) Create(ctx context.Context, userID, rawToken, userAgent, ip string, expiresAt time.Time) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{
		ID:        xid.New().String(),
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if userAgent != "" {
		rt.UserAgent = &userAgent
	}
	if ip != "" {
		rt.IP = &ip
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, user_agent, ip)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rt.ID, rt.UserID, rt.TokenHash, rt.ExpiresAt, rt.CreatedAt, rt.UserAgent, rt.IP,
	)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// GetByHash finds a non-revoked, non-expired refresh token by its hash.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, rawToken string) (*models.RefreshToken, error) {
	hash := hashToken(rawToken)
	var rt models.RefreshToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, replaced_by, user_agent, ip
		 FROM refresh_tokens
		 WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		hash,
	).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt,
		&rt.RevokedAt, &rt.ReplacedBy, &rt.UserAgent, &rt.IP,
	)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// Rotate revokes the old token and records the replacement, in one transaction.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldTokenID, newTokenID string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2, replaced_by = $3 WHERE id = $1`,
		oldTokenID, now, newTokenID,
	)
	if err != nil {
		return fmt.Errorf("rotate refresh token: %w", err)
	}
	return nil
}

// RevokeByID revokes a single refresh token.
func (r *RefreshTokenRepository) RevokeByID(ctx context.Context, tokenID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`,
		tokenID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllForUser revokes all refresh tokens for a user (logout everywhere).
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`,
		userID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired or revoked tokens older than the given cutoff.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < $1 OR (revoked_at IS NOT NULL AND revoked_at < $1)`,
		before,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// IsTokenFamilyCompromised checks if a revoked token was already used (replay detection).
// If a revoked token is presented, the entire family should be invalidated.
func (r *RefreshTokenRepository) IsTokenFamilyCompromised(ctx context.Context, rawToken string) (bool, string, error) {
	hash := hashToken(rawToken)
	var rt models.RefreshToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, revoked_at FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	).Scan(&rt.ID, &rt.UserID, &rt.RevokedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	// If the token exists and has been revoked, the family is compromised
	if rt.RevokedAt != nil {
		return true, rt.UserID, nil
	}
	return false, "", nil
}

// hashToken produces a SHA-256 hash of the raw token for storage.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
