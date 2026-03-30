package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// BackupScheduler polls the backup schedule and triggers backups at the configured cron intervals.
// It uses a simple tick-based approach (check every minute) rather than a full cron library,
// keeping dependencies minimal.
type BackupScheduler struct {
	backup *BackupService
	log    zerolog.Logger
}

// NewBackupScheduler creates a BackupScheduler.
func NewBackupScheduler(backup *BackupService, log zerolog.Logger) *BackupScheduler {
	return &BackupScheduler{backup: backup, log: log}
}

// Run starts the scheduler loop. It checks every minute whether a backup should run.
// Blocks until the context is cancelled.
func (s *BackupScheduler) Run(ctx context.Context) {
	s.log.Info().Msg("backup scheduler started")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info().Msg("backup scheduler stopped")
			return
		case t := <-ticker.C:
			s.tick(ctx, t)
		}
	}
}

// tick checks if the current minute matches the cron expression and runs a backup if so.
func (s *BackupScheduler) tick(ctx context.Context, now time.Time) {
	schedule, apiErr := s.backup.GetSchedule(ctx)
	if apiErr != nil {
		s.log.Error().Msg("scheduler: failed to read backup schedule")
		return
	}

	if !schedule.Enabled {
		return
	}

	if !cronMatches(schedule.CronExpr, now) {
		return
	}

	s.log.Info().Str("cron", schedule.CronExpr).Msg("scheduler: triggering scheduled backup")

	_, apiErr = s.backup.CreateBackup(ctx, "scheduled", "system")
	if apiErr != nil {
		s.log.Error().Str("error", apiErr.Message).Msg("scheduler: backup failed")
		return
	}

	// Rotate after creating a new backup
	deleted, err := s.backup.RotateBackups(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("scheduler: rotation failed")
	} else if deleted > 0 {
		s.log.Info().Int("deleted", deleted).Msg("scheduler: rotated expired backups")
	}
}

// cronMatches checks if a time matches a 5-field cron expression:
// minute hour day-of-month month day-of-week
func cronMatches(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}

	minute := t.Minute()
	hour := t.Hour()
	day := t.Day()
	month := int(t.Month())
	dow := int(t.Weekday()) // 0=Sunday

	return fieldMatches(fields[0], minute, 0, 59) &&
		fieldMatches(fields[1], hour, 0, 23) &&
		fieldMatches(fields[2], day, 1, 31) &&
		fieldMatches(fields[3], month, 1, 12) &&
		fieldMatches(fields[4], dow, 0, 6)
}

// fieldMatches checks if a single cron field matches a value.
// Supports: * (any), N (exact), N,M (list), N-M (range), */N (step).
func fieldMatches(field string, value, min, max int) bool {
	// Handle comma-separated alternatives
	for _, part := range strings.Split(field, ",") {
		if partMatches(part, value, min, max) {
			return true
		}
	}
	return false
}

func partMatches(part string, value, min, max int) bool {
	// Handle step: */N or N-M/N
	step := 1
	if idx := strings.Index(part, "/"); idx >= 0 {
		s, err := strconv.Atoi(part[idx+1:])
		if err != nil || s <= 0 {
			return false
		}
		step = s
		part = part[:idx]
	}

	// Wildcard
	if part == "*" {
		return (value-min)%step == 0
	}

	// Range: N-M
	if idx := strings.Index(part, "-"); idx >= 0 {
		lo, err1 := strconv.Atoi(part[:idx])
		hi, err2 := strconv.Atoi(part[idx+1:])
		if err1 != nil || err2 != nil {
			return false
		}
		if value < lo || value > hi {
			return false
		}
		return (value-lo)%step == 0
	}

	// Exact value
	n, err := strconv.Atoi(part)
	if err != nil {
		return false
	}
	return n == value
}

// FormatNextRun returns the next time the cron expression matches after the given time.
// Used for display purposes in the API.
func FormatNextRun(expr string, after time.Time) (time.Time, error) {
	// Check up to 366 days * 24 hours * 60 minutes ahead
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := t.Add(366 * 24 * time.Hour)
	for t.Before(limit) {
		if cronMatches(expr, t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no match found within a year")
}
