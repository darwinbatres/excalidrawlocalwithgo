package repository

import (
	"testing"
	"time"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

func makeBackup(id string, createdAt time.Time) models.BackupMetadata {
	return models.BackupMetadata{
		ID:        id,
		Type:      "scheduled",
		Status:    "completed",
		CreatedAt: createdAt,
	}
}

func TestSelectRetained_Empty(t *testing.T) {
	kept := selectRetained(nil, 7, 4, 6)
	if len(kept) != 0 {
		t.Errorf("expected empty set, got %d items", len(kept))
	}
}

func TestSelectRetained_DailyOnly(t *testing.T) {
	now := time.Now().UTC()
	backups := []models.BackupMetadata{
		makeBackup("b1", now.Add(-1*time.Hour)),   // today
		makeBackup("b2", now.Add(-25*time.Hour)),  // yesterday
		makeBackup("b3", now.Add(-49*time.Hour)),  // 2 days ago
		makeBackup("b4", now.Add(-200*time.Hour)), // ~8 days ago – outside daily window with keepDaily=3
	}

	kept := selectRetained(backups, 3, 0, 0)
	if !kept["b1"] || !kept["b2"] || !kept["b3"] {
		t.Errorf("expected b1, b2, b3 to be kept: %v", kept)
	}
}

func TestSelectRetained_WeeklyKeepsOnePerWeek(t *testing.T) {
	// Create backups on consecutive weeks (Monday of each week)
	// Using a fixed known date to avoid ISO week boundary issues
	base := time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC) // Monday, June 2 2025, noon

	backups := []models.BackupMetadata{
		makeBackup("w1", base),                             // Week 23
		makeBackup("w1b", base.Add(-1*time.Hour)),          // Same day, 11 AM — still Week 23
		makeBackup("w2", base.Add(-7*24*time.Hour)),        // Week 22
		makeBackup("w3", base.Add(-14*24*time.Hour)),       // Week 21
		makeBackup("w4", base.Add(-21*24*time.Hour)),       // Week 20
		makeBackup("w5", base.Add(-28*24*time.Hour)),       // Week 19
	}

	kept := selectRetained(backups, 0, 3, 0)

	// Should keep w1 (week 23), w2 (week 22), w3 (week 21) — 3 weekly
	if !kept["w1"] {
		t.Error("expected w1 to be kept")
	}
	if kept["w1b"] {
		t.Error("w1b is same week as w1, should not be counted again")
	}
	if !kept["w2"] {
		t.Error("expected w2 to be kept")
	}
	if !kept["w3"] {
		t.Error("expected w3 to be kept")
	}
	if kept["w4"] {
		t.Error("w4 should not be kept (only 3 weekly)")
	}
}

func TestSelectRetained_MonthlyKeepsOnePerMonth(t *testing.T) {
	backups := []models.BackupMetadata{
		makeBackup("m1", time.Date(2025, 6, 15, 3, 0, 0, 0, time.UTC)),  // June
		makeBackup("m1b", time.Date(2025, 6, 1, 3, 0, 0, 0, time.UTC)),  // June (second in same month)
		makeBackup("m2", time.Date(2025, 5, 15, 3, 0, 0, 0, time.UTC)),  // May
		makeBackup("m3", time.Date(2025, 4, 15, 3, 0, 0, 0, time.UTC)),  // April
		makeBackup("m4", time.Date(2025, 3, 15, 3, 0, 0, 0, time.UTC)),  // March
	}

	kept := selectRetained(backups, 0, 0, 2)

	if !kept["m1"] {
		t.Error("expected m1 (June first) to be kept")
	}
	if kept["m1b"] {
		t.Error("m1b is same month as m1, should not be counted")
	}
	if !kept["m2"] {
		t.Error("expected m2 (May) to be kept")
	}
	if kept["m3"] {
		t.Error("m3 (April) should not be kept — only 2 monthly")
	}
}

func TestSelectRetained_GFS_Combined(t *testing.T) {
	now := time.Now().UTC()
	base := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)

	// Daily: last 3 days
	// Weekly: 2 weeks
	// Monthly: 1 month
	backups := []models.BackupMetadata{
		makeBackup("d1", base.Add(-1*time.Hour)),          // today – daily
		makeBackup("d2", base.Add(-25*time.Hour)),         // yesterday – daily
		makeBackup("d3", base.Add(-49*time.Hour)),         // 2 days – daily
		makeBackup("w1", base.Add(-10*24*time.Hour)),      // ~10 days ago (different week)
		makeBackup("old", base.Add(-60*24*time.Hour)),     // ~2 months ago
	}

	kept := selectRetained(backups, 3, 2, 1)

	// d1, d2, d3 should be daily
	if !kept["d1"] || !kept["d2"] || !kept["d3"] {
		t.Errorf("expected daily backups kept: %v", kept)
	}

	// w1 may or may not be kept depending on week boundaries
	// "old" might make it as weekly/monthly depending on boundary

	// At minimum, daily ones should be there
	if len(kept) < 3 {
		t.Errorf("expected at least 3 kept, got %d", len(kept))
	}
}

func TestSelectRetained_SingleBackup(t *testing.T) {
	now := time.Now().UTC()
	backups := []models.BackupMetadata{
		makeBackup("only", now.Add(-1*time.Hour)),
	}

	kept := selectRetained(backups, 7, 4, 6)
	if !kept["only"] {
		t.Error("single backup should be kept")
	}
}
