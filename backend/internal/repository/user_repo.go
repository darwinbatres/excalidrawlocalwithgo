package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// UserRepository handles user persistence operations.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a UserRepository with the given connection pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user and returns the created user.
func (r *UserRepository) Create(ctx context.Context, email string, name *string, passwordHash *string) (*models.User, error) {
	user := &models.User{
		ID:           xid.New().String(),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.Name, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx,
		`SELECT id, email, email_verified, name, image, password_hash, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	))
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx,
		`SELECT id, email, email_verified, name, image, password_hash, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	))
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

// UpdateProfile updates the user's name and image.
func (r *UserRepository) UpdateProfile(ctx context.Context, id string, name *string, image *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = COALESCE($2, name), image = COALESCE($3, image), updated_at = $4
		 WHERE id = $1`,
		id, name, image, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}
	return nil
}

// VerifyEmail marks the user's email as verified.
func (r *UserRepository) VerifyEmail(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email_verified = $2, updated_at = $2 WHERE id = $1`,
		id, now,
	)
	if err != nil {
		return fmt.Errorf("verify user email: %w", err)
	}
	return nil
}

// CreateOrGetByOAuth finds a user by email or creates one (for OAuth sign-up).
// Returns the user and whether it was newly created.
func (r *UserRepository) CreateOrGetByOAuth(ctx context.Context, email string, name *string, image *string, emailVerified bool) (*models.User, bool, error) {
	existing, err := r.GetByEmail(ctx, email)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	now := time.Now()
	user := &models.User{
		ID:        xid.New().String(),
		Email:     email,
		Name:      name,
		Image:     image,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if emailVerified {
		user.EmailVerified = &now
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO users (id, email, email_verified, name, image, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.EmailVerified, user.Name, user.Image, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, false, err
	}

	return user, true, nil
}

func (r *UserRepository) scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.Image,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
