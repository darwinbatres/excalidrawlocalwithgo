package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// AccountRepository handles OAuth account link persistence.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository creates an AccountRepository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

// Upsert creates or updates an OAuth account link for a user.
// The unique constraint is (provider, provider_account_id).
func (r *AccountRepository) Upsert(ctx context.Context, userID, provider, providerAccountID string, accessToken, refreshToken *string, expiresAt *int) (*models.Account, error) {
	acc := &models.Account{
		ID:                xid.New().String(),
		UserID:            userID,
		Type:              "oauth",
		Provider:          provider,
		ProviderAccountID: providerAccountID,
		AccessToken:       accessToken,
		RefreshToken:      refreshToken,
		ExpiresAt:         expiresAt,
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO accounts (id, user_id, type, provider, provider_account_id, access_token, refresh_token, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (provider, provider_account_id) DO UPDATE
		 SET access_token = EXCLUDED.access_token,
		     refresh_token = EXCLUDED.refresh_token,
		     expires_at = EXCLUDED.expires_at`,
		acc.ID, acc.UserID, acc.Type, acc.Provider, acc.ProviderAccountID,
		acc.AccessToken, acc.RefreshToken, acc.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

// GetByProvider finds an account by provider and provider account ID.
func (r *AccountRepository) GetByProvider(ctx context.Context, provider, providerAccountID string) (*models.Account, error) {
	var a models.Account
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, type, provider, provider_account_id, access_token, refresh_token, expires_at, token_type, scope
		 FROM accounts
		 WHERE provider = $1 AND provider_account_id = $2`,
		provider, providerAccountID,
	).Scan(
		&a.ID, &a.UserID, &a.Type, &a.Provider, &a.ProviderAccountID,
		&a.AccessToken, &a.RefreshToken, &a.ExpiresAt, &a.TokenType, &a.Scope,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByUser returns all linked OAuth accounts for a user.
func (r *AccountRepository) ListByUser(ctx context.Context, userID string) ([]models.Account, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, provider, provider_account_id, expires_at, token_type, scope
		 FROM accounts WHERE user_id = $1 ORDER BY provider`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Type, &a.Provider, &a.ProviderAccountID,
			&a.ExpiresAt, &a.TokenType, &a.Scope,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// Delete removes an OAuth account link.
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}
