package models

import (
	"encoding/json"
	"time"
)

// BoardRole represents the permission level for board access.
type BoardRole string

const (
	BoardRoleOwner  BoardRole = "OWNER"
	BoardRoleEditor BoardRole = "EDITOR"
	BoardRoleViewer BoardRole = "VIEWER"
)

// Level returns the numeric authority level for comparison.
func (r BoardRole) Level() int {
	switch r {
	case BoardRoleOwner:
		return 100
	case BoardRoleEditor:
		return 50
	case BoardRoleViewer:
		return 25
	default:
		return 0
	}
}

// Board represents a whiteboard.
type Board struct {
	ID               string    `json:"id"`
	OrgID            string    `json:"orgId"`
	OwnerID          string    `json:"ownerId"`
	Title            string    `json:"title"`
	Description      *string   `json:"description,omitempty"`
	Tags             []string  `json:"tags"`
	IsArchived       bool      `json:"isArchived"`
	Thumbnail        *string   `json:"thumbnail,omitempty"`
	SearchContent    *string   `json:"-"`
	CurrentVersionID *string   `json:"currentVersionId,omitempty"`
	VersionNumber    int       `json:"versionNumber"`
	Etag             string    `json:"etag"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// BoardVersion represents a point-in-time snapshot of a board's content.
type BoardVersion struct {
	ID           string          `json:"id"`
	BoardID      string          `json:"boardId"`
	Version      int             `json:"version"`
	CreatedByID  string          `json:"createdById"`
	Label        *string         `json:"label,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	SceneJSON    json.RawMessage `json:"sceneJson"`
	AppStateJSON json.RawMessage `json:"appStateJson,omitempty"`
	ThumbnailURL *string         `json:"thumbnailUrl,omitempty"`
}

// BoardVersionMeta is the version info without the heavy scene data.
type BoardVersionMeta struct {
	ID          string    `json:"id"`
	BoardID     string    `json:"boardId"`
	Version     int       `json:"version"`
	CreatedByID string    `json:"createdById"`
	Label       *string   `json:"label,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// BoardPermission controls per-board access for an org member.
type BoardPermission struct {
	ID           string    `json:"id"`
	BoardID      string    `json:"boardId"`
	MembershipID string    `json:"membershipId"`
	Role         BoardRole `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// BoardWithVersion is a board combined with its latest version data.
type BoardWithVersion struct {
	Board
	LatestVersion *BoardVersion `json:"latestVersion,omitempty"`
}
