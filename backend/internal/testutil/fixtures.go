package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// UserFixture creates a user directly in the database.
func UserFixture(t *testing.T, pool *pgxpool.Pool, email string) *models.User {
	t.Helper()
	var u models.User
	id := xid.New().String()
	hash := "$2a$12$LJ3m4ys3Lk0TSwHilbpVwuGh8jGn/cGzkVYJlIkTiYQKPmFqVOxMi" // bcrypt("password123")
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (id, email, password_hash, name, email_verified)
		 VALUES ($1, $2, $3, $4, NOW())
		 RETURNING id, email, name, image, email_verified, created_at, updated_at`,
		id, email, hash, strPtr(email),
	).Scan(&u.ID, &u.Email, &u.Name, &u.Image, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create user fixture: %v", err)
	}
	return &u
}

// OrgFixture creates an organization directly in the database.
func OrgFixture(t *testing.T, pool *pgxpool.Pool, name, slug string) *models.Organization {
	t.Helper()
	var org models.Organization
	id := xid.New().String()
	err := pool.QueryRow(context.Background(),
		`INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)
		 RETURNING id, name, slug, created_at, updated_at`,
		id, name, slug,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create org fixture: %v", err)
	}
	return &org
}

// MembershipFixture creates a membership directly in the database.
func MembershipFixture(t *testing.T, pool *pgxpool.Pool, orgID, userID string, role models.OrgRole) *models.Membership {
	t.Helper()
	var m models.Membership
	id := xid.New().String()
	err := pool.QueryRow(context.Background(),
		`INSERT INTO memberships (id, org_id, user_id, role) VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, user_id, role, created_at`,
		id, orgID, userID, string(role),
	).Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt)
	if err != nil {
		t.Fatalf("failed to create membership fixture: %v", err)
	}
	return &m
}

// BoardFixture creates a board directly in the database.
func BoardFixture(t *testing.T, pool *pgxpool.Pool, orgID, ownerID, title string) *models.Board {
	t.Helper()
	var b models.Board
	id := xid.New().String()
	etag := fmt.Sprintf("etag-%d", time.Now().UnixNano())
	err := pool.QueryRow(context.Background(),
		`INSERT INTO boards (id, org_id, owner_id, title, etag, version_number)
		 VALUES ($1, $2, $3, $4, $5, 0)
		 RETURNING id, org_id, owner_id, title, description, tags, is_archived, etag, version_number, current_version_id, created_at, updated_at`,
		id, orgID, ownerID, title, etag,
	).Scan(&b.ID, &b.OrgID, &b.OwnerID, &b.Title, &b.Description, &b.Tags, &b.IsArchived,
		&b.Etag, &b.VersionNumber, &b.CurrentVersionID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create board fixture: %v", err)
	}
	return &b
}

// AuditFixture creates an audit event directly in the database.
func AuditFixture(t *testing.T, pool *pgxpool.Pool, orgID string, actorID *string, action, targetType, targetID string) {
	t.Helper()
	id := xid.New().String()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO audit_events (id, org_id, actor_id, action, target_type, target_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, orgID, actorID, action, targetType, targetID,
	)
	if err != nil {
		t.Fatalf("failed to create audit fixture: %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}
