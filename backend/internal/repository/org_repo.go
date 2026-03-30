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

// OrgRepository handles organization persistence operations.
type OrgRepository struct {
	pool *pgxpool.Pool
}

// NewOrgRepository creates an OrgRepository with the given connection pool.
func NewOrgRepository(pool *pgxpool.Pool) *OrgRepository {
	return &OrgRepository{pool: pool}
}

// OrgWithCounts includes member and board counts for list responses.
type OrgWithCounts struct {
	models.Organization
	Role        models.OrgRole `json:"role"`
	MemberCount int            `json:"memberCount"`
	BoardCount  int            `json:"boardCount"`
}

// Create inserts a new organization.
func (r *OrgRepository) Create(ctx context.Context, name, slug string) (*models.Organization, error) {
	org := &models.Organization{
		ID:        xid.New().String(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// GetByID retrieves an organization by ID.
func (r *OrgRepository) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	var org models.Organization
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, created_at, updated_at
		 FROM organizations WHERE id = $1`, id,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// SlugExists checks whether an organization slug is already taken.
func (r *OrgRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1)`, slug,
	).Scan(&exists)
	return exists, err
}

// ListByUser returns all organizations the user belongs to with counts.
func (r *OrgRepository) ListByUser(ctx context.Context, userID string) ([]OrgWithCounts, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT o.id, o.name, o.slug, o.created_at, o.updated_at,
		        m.role,
		        (SELECT COUNT(*) FROM memberships WHERE org_id = o.id) AS member_count,
		        (SELECT COUNT(*) FROM boards WHERE org_id = o.id AND is_archived = false) AS board_count
		 FROM organizations o
		 JOIN memberships m ON m.org_id = o.id AND m.user_id = $1
		 ORDER BY o.name ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []OrgWithCounts
	for rows.Next() {
		var oc OrgWithCounts
		if err := rows.Scan(
			&oc.ID, &oc.Name, &oc.Slug, &oc.CreatedAt, &oc.UpdatedAt,
			&oc.Role, &oc.MemberCount, &oc.BoardCount,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, oc)
	}
	return orgs, rows.Err()
}

// Update renames an organization.
func (r *OrgRepository) Update(ctx context.Context, id, name string) (*models.Organization, error) {
	var org models.Organization
	err := r.pool.QueryRow(ctx,
		`UPDATE organizations SET name = $2, updated_at = $3
		 WHERE id = $1
		 RETURNING id, name, slug, created_at, updated_at`,
		id, name, time.Now(),
	).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// Delete removes an organization by ID. Relies on DB cascade for memberships.
func (r *OrgRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete org: %w", err)
	}
	return nil
}

// BoardCount returns the total number of boards in an organization.
func (r *OrgRepository) BoardCount(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM boards WHERE org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

// CreateInTx inserts a new organization within an existing transaction.
func (r *OrgRepository) CreateInTx(ctx context.Context, tx pgx.Tx, name, slug string) (*models.Organization, error) {
	org := &models.Organization{
		ID:        xid.New().String(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := tx.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return org, nil
}
