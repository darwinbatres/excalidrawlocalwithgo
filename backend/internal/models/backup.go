package models

import "time"

// BackupMetadata tracks database backup files.
type BackupMetadata struct {
	ID          string     `json:"id"`
	Filename    string     `json:"filename"`
	SizeBytes   int64      `json:"sizeBytes"`
	StorageKey  string     `json:"storageKey"`
	Type        string     `json:"type"` // "manual" or "scheduled"
	Status      string     `json:"status"` // "in_progress", "completed", "failed"
	DurationMs  *int       `json:"durationMs,omitempty"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// BackupSchedule holds the automated backup configuration.
type BackupSchedule struct {
	ID          string    `json:"id"`
	Enabled     bool      `json:"enabled"`
	CronExpr    string    `json:"cronExpr"`
	KeepDaily   int       `json:"keepDaily"`
	KeepWeekly  int       `json:"keepWeekly"`
	KeepMonthly int       `json:"keepMonthly"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
