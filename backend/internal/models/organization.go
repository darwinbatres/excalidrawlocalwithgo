package models

import "time"

// OrgRole represents the role hierarchy within an organization.
type OrgRole string

const (
	OrgRoleOwner  OrgRole = "OWNER"
	OrgRoleAdmin  OrgRole = "ADMIN"
	OrgRoleMember OrgRole = "MEMBER"
	OrgRoleViewer OrgRole = "VIEWER"
)

// Level returns the numeric authority level for comparison.
func (r OrgRole) Level() int {
	switch r {
	case OrgRoleOwner:
		return 100
	case OrgRoleAdmin:
		return 75
	case OrgRoleMember:
		return 50
	case OrgRoleViewer:
		return 25
	default:
		return 0
	}
}

// Organization represents a tenant/workspace.
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Membership links a user to an organization with a role.
type Membership struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"orgId"`
	UserID    string    `json:"userId"`
	Role      OrgRole   `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MembershipWithUser includes user details for member listings.
type MembershipWithUser struct {
	Membership
	User UserPublic `json:"user"`
}
