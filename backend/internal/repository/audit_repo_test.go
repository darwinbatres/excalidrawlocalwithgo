package repository

import (
	"testing"
	"time"
)

func TestAuditQueryParams_Defaults(t *testing.T) {
	params := AuditQueryParams{
		OrgID: "org-1",
	}

	if params.OrgID != "org-1" {
		t.Errorf("expected org-1, got %s", params.OrgID)
	}
	if params.Limit != 0 {
		t.Errorf("expected default limit 0, got %d", params.Limit)
	}
	if params.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", params.Offset)
	}
	if params.StartDate != nil {
		t.Error("expected nil StartDate")
	}
	if params.EndDate != nil {
		t.Error("expected nil EndDate")
	}
}

func TestAuditQueryParams_AllFilters(t *testing.T) {
	now := time.Now()
	params := AuditQueryParams{
		OrgID:      "org-1",
		ActorID:    "user-1",
		Action:     "board.created",
		TargetType: "board",
		TargetID:   "board-1",
		StartDate:  &now,
		EndDate:    &now,
		Limit:      50,
		Offset:     10,
	}

	if params.ActorID != "user-1" {
		t.Errorf("expected user-1, got %s", params.ActorID)
	}
	if params.Action != "board.created" {
		t.Errorf("expected board.created, got %s", params.Action)
	}
	if params.TargetType != "board" {
		t.Errorf("expected board, got %s", params.TargetType)
	}
	if params.TargetID != "board-1" {
		t.Errorf("expected board-1, got %s", params.TargetID)
	}
	if params.Limit != 50 {
		t.Errorf("expected 50, got %d", params.Limit)
	}
	if params.Offset != 10 {
		t.Errorf("expected 10, got %d", params.Offset)
	}
}

func TestSystemCounts_ZeroValues(t *testing.T) {
	c := SystemCounts{}
	if c.Users != 0 || c.Organizations != 0 || c.Boards != 0 || c.BoardsActive != 0 ||
		c.BoardsArchived != 0 || c.BoardVersions != 0 || c.BoardAssets != 0 ||
		c.AuditEvents != 0 || c.Memberships != 0 || c.ShareLinks != 0 ||
		c.ShareLinksActive != 0 || c.ShareLinksExpired != 0 || c.RefreshTokens != 0 ||
		c.RefreshTokensActive != 0 || c.Backups != 0 || c.BackupsCompleted != 0 ||
		c.BackupsFailed != 0 || c.BackupsInProgress != 0 {
		t.Error("expected all zero values")
	}
}

func TestAuditStats_EmptyByAction(t *testing.T) {
	stats := AuditStats{
		TotalEvents: 100,
		ByAction:    nil,
	}
	if stats.TotalEvents != 100 {
		t.Errorf("expected 100, got %d", stats.TotalEvents)
	}
	if stats.ByAction != nil {
		t.Errorf("expected nil ByAction, got %v", stats.ByAction)
	}
}

func TestAuditQueryResult_Empty(t *testing.T) {
	result := AuditQueryResult{
		Total: 0,
	}
	if result.Total != 0 {
		t.Errorf("expected 0, got %d", result.Total)
	}
	if result.Events != nil {
		t.Errorf("expected nil events, got %v", result.Events)
	}
}
