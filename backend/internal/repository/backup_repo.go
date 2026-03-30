package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// BackupRepository handles backup metadata and schedule persistence.
type BackupRepository struct {
	pool *pgxpool.Pool
}

// NewBackupRepository creates a BackupRepository.
func NewBackupRepository(pool *pgxpool.Pool) *BackupRepository {
	return &BackupRepository{pool: pool}
}

// CreateMetadata inserts a new backup metadata record (status = in_progress).
func (r *BackupRepository) CreateMetadata(ctx context.Context, backupType string) (*models.BackupMetadata, error) {
	m := &models.BackupMetadata{
		ID:        xid.New().String(),
		Type:      backupType,
		Status:    "in_progress",
		CreatedAt: time.Now().UTC(),
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO backup_metadata (id, filename, size_bytes, storage_key, type, status, created_at)
		 VALUES ($1, '', 0, '', $2, $3, $4)`,
		m.ID, m.Type, m.Status, m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// CompleteMetadata marks a backup as completed with file details.
func (r *BackupRepository) CompleteMetadata(ctx context.Context, id, filename, storageKey string, sizeBytes int64, durationMs int) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE backup_metadata
		 SET filename = $2, storage_key = $3, size_bytes = $4, duration_ms = $5,
		     status = 'completed', completed_at = $6
		 WHERE id = $1`,
		id, filename, storageKey, sizeBytes, durationMs, now,
	)
	if err != nil {
		return fmt.Errorf("complete backup metadata: %w", err)
	}
	return nil
}

// FailMetadata marks a backup as failed with an error message.
func (r *BackupRepository) FailMetadata(ctx context.Context, id, errMsg string, durationMs int) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE backup_metadata
		 SET status = 'failed', error = $2, duration_ms = $3, completed_at = $4
		 WHERE id = $1`,
		id, errMsg, durationMs, now,
	)
	if err != nil {
		return fmt.Errorf("fail backup metadata: %w", err)
	}
	return nil
}

// GetMetadata returns a single backup metadata record.
func (r *BackupRepository) GetMetadata(ctx context.Context, id string) (*models.BackupMetadata, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, filename, size_bytes, storage_key, type, status,
		        duration_ms, error, created_at, completed_at
		 FROM backup_metadata WHERE id = $1`, id,
	)
	return scanBackupMetadata(row)
}

// ListMetadata returns paginated backup records ordered by most recent first.
func (r *BackupRepository) ListMetadata(ctx context.Context, limit, offset int) ([]models.BackupMetadata, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM backup_metadata`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, filename, size_bytes, storage_key, type, status,
		        duration_ms, error, created_at, completed_at
		 FROM backup_metadata ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []models.BackupMetadata
	for rows.Next() {
		m, err := scanBackupMetadata(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, *m)
	}
	return results, total, nil
}

// DeleteMetadata removes a backup metadata record.
func (r *BackupRepository) DeleteMetadata(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM backup_metadata WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete backup metadata: %w", err)
	}
	return nil
}

// ListExpiredForRotation returns completed backups that should be pruned based on retention policy.
// keepDaily: number of daily backups to keep (most recent N days)
// keepWeekly: number of weekly backups to keep
// keepMonthly: number of monthly backups to keep
// Returns IDs + storage keys of backups to delete.
func (r *BackupRepository) ListExpiredForRotation(ctx context.Context, keepDaily, keepWeekly, keepMonthly int) ([]models.BackupMetadata, error) {
	// Get all completed backups ordered by time
	rows, err := r.pool.Query(ctx,
		`SELECT id, filename, size_bytes, storage_key, type, status,
		        duration_ms, error, created_at, completed_at
		 FROM backup_metadata WHERE status = 'completed'
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []models.BackupMetadata
	for rows.Next() {
		m, err := scanBackupMetadata(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, *m)
	}

	kept := selectRetained(all, keepDaily, keepWeekly, keepMonthly)
	var expired []models.BackupMetadata
	for _, m := range all {
		if !kept[m.ID] {
			expired = append(expired, m)
		}
	}
	return expired, nil
}

// selectRetained applies a GFS (Grandfather-Father-Son) retention policy.
// It returns a set of backup IDs to keep.
func selectRetained(backups []models.BackupMetadata, keepDaily, keepWeekly, keepMonthly int) map[string]bool {
	kept := make(map[string]bool)
	now := time.Now().UTC()

	// Keep most recent N daily backups
	dailyCount := 0
	for _, b := range backups {
		if dailyCount >= keepDaily {
			break
		}
		if now.Sub(b.CreatedAt) <= time.Duration(keepDaily)*24*time.Hour {
			kept[b.ID] = true
			dailyCount++
		}
	}

	// Keep one backup per week for keepWeekly weeks
	weeklyCount := 0
	var lastWeek int
	for _, b := range backups {
		if weeklyCount >= keepWeekly {
			break
		}
		y, w := b.CreatedAt.ISOWeek()
		weekKey := y*100 + w
		if weekKey != lastWeek {
			kept[b.ID] = true
			lastWeek = weekKey
			weeklyCount++
		}
	}

	// Keep one backup per month for keepMonthly months
	monthlyCount := 0
	var lastMonth int
	for _, b := range backups {
		if monthlyCount >= keepMonthly {
			break
		}
		monthKey := b.CreatedAt.Year()*100 + int(b.CreatedAt.Month())
		if monthKey != lastMonth {
			kept[b.ID] = true
			lastMonth = monthKey
			monthlyCount++
		}
	}

	return kept
}

// GetSchedule returns the (singleton) backup schedule.
func (r *BackupRepository) GetSchedule(ctx context.Context) (*models.BackupSchedule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, enabled, cron_expr, keep_daily, keep_weekly, keep_monthly, updated_at
		 FROM backup_schedule WHERE id = 'default'`,
	)
	var s models.BackupSchedule
	err := row.Scan(&s.ID, &s.Enabled, &s.CronExpr, &s.KeepDaily, &s.KeepWeekly, &s.KeepMonthly, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateSchedule updates the backup schedule settings.
func (r *BackupRepository) UpdateSchedule(ctx context.Context, enabled bool, cronExpr string, keepDaily, keepWeekly, keepMonthly int) (*models.BackupSchedule, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE backup_schedule
		 SET enabled = $1, cron_expr = $2, keep_daily = $3, keep_weekly = $4,
		     keep_monthly = $5, updated_at = $6
		 WHERE id = 'default'
		 RETURNING id, enabled, cron_expr, keep_daily, keep_weekly, keep_monthly, updated_at`,
		enabled, cronExpr, keepDaily, keepWeekly, keepMonthly, now,
	)
	var s models.BackupSchedule
	err := row.Scan(&s.ID, &s.Enabled, &s.CronExpr, &s.KeepDaily, &s.KeepWeekly, &s.KeepMonthly, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// scannable is satisfied by both pgx.Row and pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

func scanBackupMetadata(row scannable) (*models.BackupMetadata, error) {
	var m models.BackupMetadata
	err := row.Scan(
		&m.ID, &m.Filename, &m.SizeBytes, &m.StorageKey, &m.Type, &m.Status,
		&m.DurationMs, &m.Error, &m.CreatedAt, &m.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
