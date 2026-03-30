package repository

import (
	"context"
	"encoding/json"
	"fmt"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/xid"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

// AuditRepository handles audit event persistence.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates an AuditRepository.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Log records an audit event. This is fire-and-forget friendly — errors are returned
// but callers may choose to log and continue rather than failing the request.
func (r *AuditRepository) Log(ctx context.Context, orgID string, actorID *string, action, targetType, targetID string, ip, userAgent *string, metadata map[string]any) error {
	var metaJSON json.RawMessage
	if metadata != nil {
		var err error
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("log audit event: marshal metadata: %w", err)
		}
	}

	// Pass NULL for org_id when empty (e.g. login/registration events).
	var orgIDParam any
	if orgID != "" {
		orgIDParam = orgID
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_events (id, org_id, actor_id, action, target_type, target_id, ip, user_agent, metadata, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		xid.New().String(), orgIDParam, actorID, action, targetType, targetID,
		ip, userAgent, metaJSON, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("log audit event: %w", err)
	}
	return nil
}

// AuditQueryParams defines filters for listing audit events.
type AuditQueryParams struct {
	OrgID      string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

// AuditQueryResult is a paginated audit query result.
type AuditQueryResult struct {
	Events []models.AuditEvent `json:"events"`
	Total  int64               `json:"total"`
}

// Query returns audit events matching the filter criteria with pagination.
func (r *AuditRepository) Query(ctx context.Context, params AuditQueryParams) (*AuditQueryResult, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
	args = append(args, params.OrgID)
	argIdx++

	if params.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, params.ActorID)
		argIdx++
	}

	if params.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, params.Action)
		argIdx++
	}

	if params.TargetType != "" {
		conditions = append(conditions, fmt.Sprintf("target_type = $%d", argIdx))
		args = append(args, params.TargetType)
		argIdx++
	}

	if params.TargetID != "" {
		conditions = append(conditions, fmt.Sprintf("target_id = $%d", argIdx))
		args = append(args, params.TargetID)
		argIdx++
	}

	if params.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *params.StartDate)
		argIdx++
	}

	if params.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *params.EndDate)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count total matches
	var total int64
	countQuery := "SELECT COUNT(*) FROM audit_events " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch page
	dataQuery := fmt.Sprintf(
		`SELECT id, org_id, actor_id, action, target_type, target_id, ip, user_agent, metadata, created_at
		 FROM audit_events %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, params.Limit, params.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.AuditEvent, 0)
	for rows.Next() {
		var e models.AuditEvent
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.ActorID, &e.Action, &e.TargetType, &e.TargetID,
			&e.IP, &e.UserAgent, &e.Metadata, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &AuditQueryResult{Events: events, Total: total}, nil
}

// AuditActionCount represents event count grouped by action.
type AuditActionCount struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}

// AuditStats represents aggregated audit statistics.
type AuditStats struct {
	TotalEvents int64              `json:"totalEvents"`
	ByAction    []AuditActionCount `json:"byAction"`
}

// Stats returns aggregated audit statistics for an org over a time range.
func (r *AuditRepository) Stats(ctx context.Context, orgID string, since time.Time) (*AuditStats, error) {
	var totalEvents int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_events WHERE org_id = $1 AND created_at >= $2`,
		orgID, since,
	).Scan(&totalEvents)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT action, COUNT(*) AS cnt FROM audit_events
		 WHERE org_id = $1 AND created_at >= $2
		 GROUP BY action ORDER BY cnt DESC`,
		orgID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var byAction []AuditActionCount
	for rows.Next() {
		var ac AuditActionCount
		if err := rows.Scan(&ac.Action, &ac.Count); err != nil {
			return nil, err
		}
		byAction = append(byAction, ac)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if byAction == nil {
		byAction = []AuditActionCount{}
	}
	return &AuditStats{TotalEvents: totalEvents, ByAction: byAction}, nil
}

// GlobalCRUDBreakdown returns audit event counts grouped by action across all orgs.
func (r *AuditRepository) GlobalCRUDBreakdown(ctx context.Context) (*CRUDBreakdown, error) {
	var totalEvents int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_events`,
	).Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("global crud breakdown count: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT action, COUNT(*) AS cnt FROM audit_events
		 GROUP BY action ORDER BY cnt DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("global crud breakdown: %w", err)
	}
	defer rows.Close()

	var byAction []AuditActionCount
	for rows.Next() {
		var ac AuditActionCount
		if err := rows.Scan(&ac.Action, &ac.Count); err != nil {
			return nil, err
		}
		byAction = append(byAction, ac)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if byAction == nil {
		byAction = []AuditActionCount{}
	}
	return &CRUDBreakdown{TotalEvents: totalEvents, ByAction: byAction}, nil
}

// OrgCounts returns per-table row counts scoped to a single organization.
type OrgCounts struct {
	Boards         int64 `json:"boards"`
	BoardsActive   int64 `json:"boardsActive"`
	BoardsArchived int64 `json:"boardsArchived"`
	BoardVersions  int64 `json:"boardVersions"`
	BoardAssets    int64 `json:"boardAssets"`
	Members        int64 `json:"members"`
	AuditEvents    int64 `json:"auditEvents"`
	ShareLinks        int64 `json:"shareLinks"`
	ShareLinksActive  int64 `json:"shareLinksActive"`
	ShareLinksExpired int64 `json:"shareLinksExpired"`
}

// CountAllByOrg returns row counts for tables scoped to a specific organization.
// Also returns the organization name.
func (r *AuditRepository) CountAllByOrg(ctx context.Context, orgID string) (string, *OrgCounts, error) {
	var orgName string
	var c OrgCounts
	err := r.pool.QueryRow(ctx,
		`SELECT
			(SELECT COALESCE(name, '') FROM organizations WHERE id = $1),
			(SELECT COUNT(*) FROM boards WHERE org_id = $1),
			(SELECT COUNT(*) FROM boards WHERE org_id = $1 AND is_archived = false),
			(SELECT COUNT(*) FROM boards WHERE org_id = $1 AND is_archived = true),
			(SELECT COUNT(*) FROM board_versions WHERE board_id IN (SELECT id FROM boards WHERE org_id = $1)),
			(SELECT COUNT(*) FROM board_assets WHERE board_id IN (SELECT id FROM boards WHERE org_id = $1)),
			(SELECT COUNT(*) FROM memberships WHERE org_id = $1),
			(SELECT COUNT(*) FROM audit_events WHERE org_id = $1),
			(SELECT COUNT(*) FROM share_links WHERE board_id IN (SELECT id FROM boards WHERE org_id = $1)),
			(SELECT COUNT(*) FROM share_links WHERE board_id IN (SELECT id FROM boards WHERE org_id = $1) AND (expires_at IS NULL OR expires_at > NOW())),
			(SELECT COUNT(*) FROM share_links WHERE board_id IN (SELECT id FROM boards WHERE org_id = $1) AND expires_at IS NOT NULL AND expires_at <= NOW())`,
		orgID,
	).Scan(
		&orgName,
		&c.Boards, &c.BoardsActive, &c.BoardsArchived,
		&c.BoardVersions, &c.BoardAssets,
		&c.Members, &c.AuditEvents,
		&c.ShareLinks, &c.ShareLinksActive, &c.ShareLinksExpired,
	)
	if err != nil {
		return "", nil, fmt.Errorf("count all by org: %w", err)
	}
	return orgName, &c, nil
}

// OrgCRUDBreakdown returns audit event counts grouped by action for a specific org.
func (r *AuditRepository) OrgCRUDBreakdown(ctx context.Context, orgID string) (*CRUDBreakdown, error) {
	var totalEvents int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_events WHERE org_id = $1`, orgID,
	).Scan(&totalEvents)
	if err != nil {
		return nil, fmt.Errorf("org crud breakdown count: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT action, COUNT(*) AS cnt FROM audit_events
		 WHERE org_id = $1 GROUP BY action ORDER BY cnt DESC`, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("org crud breakdown: %w", err)
	}
	defer rows.Close()

	var byAction []AuditActionCount
	for rows.Next() {
		var ac AuditActionCount
		if err := rows.Scan(&ac.Action, &ac.Count); err != nil {
			return nil, err
		}
		byAction = append(byAction, ac)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if byAction == nil {
		byAction = []AuditActionCount{}
	}
	return &CRUDBreakdown{TotalEvents: totalEvents, ByAction: byAction}, nil
}

// SystemCounts returns per-table row counts for the system stats endpoint.
// Includes granular breakdowns (active/archived/expired) so the frontend
// can display accurate, non-misleading numbers.
type SystemCounts struct {
	Users          int64 `json:"users"`
	Organizations  int64 `json:"organizations"`
	Boards         int64 `json:"boards"`
	BoardsActive   int64 `json:"boardsActive"`
	BoardsArchived int64 `json:"boardsArchived"`
	BoardVersions  int64 `json:"boardVersions"`
	BoardAssets    int64 `json:"boardAssets"`
	AuditEvents    int64 `json:"auditEvents"`
	Memberships    int64 `json:"memberships"`
	ShareLinks        int64 `json:"shareLinks"`
	ShareLinksActive  int64 `json:"shareLinksActive"`
	ShareLinksExpired int64 `json:"shareLinksExpired"`
	Accounts       int64 `json:"accounts"`
	RefreshTokens       int64 `json:"refreshTokens"`
	RefreshTokensActive int64 `json:"refreshTokensActive"`
	Backups            int64 `json:"backups"`
	BackupsCompleted   int64 `json:"backupsCompleted"`
	BackupsFailed      int64 `json:"backupsFailed"`
	BackupsInProgress  int64 `json:"backupsInProgress"`
}

// TableSizeInfo describes the storage footprint of a single database table.
type TableSizeInfo struct {
	Table      string `json:"table"`
	TotalBytes int64  `json:"totalBytes"`
	DataBytes  int64  `json:"dataBytes"`
	IndexBytes int64  `json:"indexBytes"`
	RowCount   int64  `json:"rowCount"`
}

// DatabaseStats contains comprehensive database metrics.
type DatabaseStats struct {
	DatabaseSizeBytes int64           `json:"databaseSizeBytes"`
	Tables            []TableSizeInfo `json:"tables"`
	ActiveConnections int64           `json:"activeConnections"`
	MaxConnections    int64           `json:"maxConnections"`
	CacheHitRatio     float64         `json:"cacheHitRatio"`
	Uptime            string          `json:"uptime"`
}

// PoolStats contains pgxpool connection-pool metrics.
type PoolStats struct {
	AcquireCount         int64 `json:"acquireCount"`
	AcquiredConns        int32 `json:"acquiredConns"`
	IdleConns            int32 `json:"idleConns"`
	TotalConns           int32 `json:"totalConns"`
	ConstructingConns    int32 `json:"constructingConns"`
	MaxConns             int32 `json:"maxConns"`
	EmptyAcquireCount    int64 `json:"emptyAcquireCount"`
	CanceledAcquireCount int64 `json:"canceledAcquireCount"`
}

// RuntimeStats contains Go runtime metrics.
type RuntimeStats struct {
	Goroutines  int    `json:"goroutines"`
	HeapAlloc   uint64 `json:"heapAlloc"`
	HeapSys     uint64 `json:"heapSys"`
	HeapInuse   uint64 `json:"heapInuse"`
	StackInuse  uint64 `json:"stackInuse"`
	GCPauseNs   uint64 `json:"gcPauseNs"`
	NumGC       uint32 `json:"numGC"`
	NumCPU      int    `json:"numCPU"`
	GoVersion   string `json:"goVersion"`
}

// ProcessStats contains OS-level process metrics.
type ProcessStats struct {
	PID           int    `json:"pid"`
	RSS           uint64 `json:"rss"`
	OpenFDs       int    `json:"openFDs"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
	StartTime     string `json:"startTime"`
}

// ContainerStats contains cgroup-aware container metrics (Linux only).
type ContainerStats struct {
	IsContainer        bool    `json:"isContainer"`
	MemoryLimitBytes   int64   `json:"memoryLimitBytes,omitempty"`
	MemoryUsageBytes   int64   `json:"memoryUsageBytes,omitempty"`
	MemoryUsagePercent float64 `json:"memoryUsagePercent,omitempty"`
	CPUQuota           int64   `json:"cpuQuota,omitempty"`
	CPUPeriod          int64   `json:"cpuPeriod,omitempty"`
	EffectiveCPUs      float64 `json:"effectiveCpus,omitempty"`
	ContainerID        string  `json:"containerId,omitempty"`
	// CPU usage metrics from cgroup cpu.stat
	CPUUsageUsec    int64 `json:"cpuUsageUsec,omitempty"`
	ThrottledUsec   int64 `json:"throttledUsec,omitempty"`
	ThrottledPeriods int64 `json:"throttledPeriods,omitempty"`
	// Memory detail from cgroup memory.stat
	MemoryCacheBytes int64 `json:"memoryCacheBytes,omitempty"`
	SwapUsageBytes   int64 `json:"swapUsageBytes,omitempty"`
	OOMKills         int64 `json:"oomKills,omitempty"`
	// PID limits from cgroup pids controller
	PIDsCurrent int64 `json:"pidsCurrent,omitempty"`
	PIDsLimit   int64 `json:"pidsLimit,omitempty"`
	// Network I/O from /proc/self/net/dev
	NetworkRxBytes   uint64 `json:"networkRxBytes,omitempty"`
	NetworkTxBytes   uint64 `json:"networkTxBytes,omitempty"`
	NetworkRxPackets uint64 `json:"networkRxPackets,omitempty"`
	NetworkTxPackets uint64 `json:"networkTxPackets,omitempty"`
}

// S3StorageStats contains per-bucket storage metrics.
type S3StorageStats struct {
	Buckets    []BucketInfo `json:"buckets"`
	TotalBytes int64        `json:"totalBytes"`
	TotalObjects int64      `json:"totalObjects"`
}

// BucketInfo describes storage usage for a single S3 bucket.
type BucketInfo struct {
	Bucket      string `json:"bucket"`
	ObjectCount int64  `json:"objectCount"`
	TotalBytes  int64  `json:"totalBytes"`
	LargestBytes int64 `json:"largestBytes"`
}

// BuildInfo contains build metadata.
type BuildInfo struct {
	Version   string `json:"version"`
	CommitSHA string `json:"commitSHA"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
}

// CRUDBreakdown contains counts of CRUD operations grouped by audit action.
type CRUDBreakdown struct {
	ByAction    []AuditActionCount `json:"byAction"`
	TotalEvents int64              `json:"totalEvents"`
}

// BackupStats contains backup-related metrics.
type BackupStats struct {
	TotalBackups    int    `json:"totalBackups"`
	LastBackupAt    string `json:"lastBackupAt,omitempty"`
	LastBackupSize  int64  `json:"lastBackupSizeBytes,omitempty"`
	LastBackupStatus string `json:"lastBackupStatus,omitempty"`
	ScheduleEnabled bool   `json:"scheduleEnabled"`
	ScheduleCron    string `json:"scheduleCron,omitempty"`
}

// LogSummary contains log-level counts from the in-memory ring buffer.
type LogSummary struct {
	Debug int `json:"debug"`
	Info  int `json:"info"`
	Warn  int `json:"warn"`
	Error int `json:"error"`
	Fatal int `json:"fatal"`
	Total int `json:"total"`
}

// SystemStatsResult combines row counts, database metrics, pool stats, runtime info,
// S3 storage, process stats, and build info.
type SystemStatsResult struct {
	Counts        SystemCounts   `json:"counts"`
	Database      DatabaseStats  `json:"database"`
	Pool          PoolStats      `json:"pool"`
	Runtime       RuntimeStats   `json:"runtime"`
	Storage       S3StorageStats `json:"storage"`
	Process       ProcessStats   `json:"process"`
	Container     ContainerStats `json:"container"`
	Build         BuildInfo      `json:"build"`
	Requests      any            `json:"requests,omitempty"`
	BruteForce    any            `json:"bruteForce,omitempty"`
	CRUDBreakdown *CRUDBreakdown `json:"crudBreakdown,omitempty"`
	BackupInfo    *BackupStats   `json:"backupInfo,omitempty"`
	Logs          *LogSummary    `json:"logs,omitempty"`
}

// OrgStatsResult contains org-scoped counts and CRUD breakdown.
type OrgStatsResult struct {
	OrgID         string         `json:"orgId"`
	OrgName       string         `json:"orgName"`
	Counts        OrgCounts      `json:"counts"`
	CRUDBreakdown *CRUDBreakdown `json:"crudBreakdown,omitempty"`
}

// CountAll returns row counts for all major tables with granular breakdowns.
// Active/archived/expired splits let the frontend display accurate stats.
func (r *AuditRepository) CountAll(ctx context.Context) (*SystemCounts, error) {
	var c SystemCounts
	err := r.pool.QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM organizations),
			(SELECT COUNT(*) FROM boards),
			(SELECT COUNT(*) FROM boards WHERE is_archived = false),
			(SELECT COUNT(*) FROM boards WHERE is_archived = true),
			(SELECT COUNT(*) FROM board_versions),
			(SELECT COUNT(*) FROM board_assets),
			(SELECT COUNT(*) FROM audit_events),
			(SELECT COUNT(*) FROM memberships),
			(SELECT COUNT(*) FROM share_links),
			(SELECT COUNT(*) FROM share_links WHERE expires_at IS NULL OR expires_at > NOW()),
			(SELECT COUNT(*) FROM share_links WHERE expires_at IS NOT NULL AND expires_at <= NOW()),
			(SELECT COUNT(*) FROM accounts),
			(SELECT COUNT(*) FROM refresh_tokens),
			(SELECT COUNT(*) FROM refresh_tokens WHERE revoked_at IS NULL AND expires_at > NOW()),
			(SELECT COUNT(*) FROM backup_metadata),
			(SELECT COUNT(*) FROM backup_metadata WHERE status = 'completed'),
			(SELECT COUNT(*) FROM backup_metadata WHERE status = 'failed'),
			(SELECT COUNT(*) FROM backup_metadata WHERE status = 'in_progress')`,
	).Scan(
		&c.Users, &c.Organizations,
		&c.Boards, &c.BoardsActive, &c.BoardsArchived,
		&c.BoardVersions, &c.BoardAssets, &c.AuditEvents, &c.Memberships,
		&c.ShareLinks, &c.ShareLinksActive, &c.ShareLinksExpired,
		&c.Accounts,
		&c.RefreshTokens, &c.RefreshTokensActive,
		&c.Backups, &c.BackupsCompleted, &c.BackupsFailed, &c.BackupsInProgress,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetDatabaseStats returns comprehensive database-level metrics.
func (r *AuditRepository) GetDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Database size
	err := r.pool.QueryRow(ctx,
		`SELECT pg_database_size(current_database())`,
	).Scan(&stats.DatabaseSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("database size: %w", err)
	}

	// Connection info
	err = r.pool.QueryRow(ctx,
		`SELECT
			(SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()),
			(SELECT setting::bigint FROM pg_settings WHERE name = 'max_connections')`,
	).Scan(&stats.ActiveConnections, &stats.MaxConnections)
	if err != nil {
		return nil, fmt.Errorf("connection stats: %w", err)
	}

	// Cache hit ratio
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(
			sum(heap_blks_hit)::float / NULLIF(sum(heap_blks_hit) + sum(heap_blks_read), 0),
			0
		) FROM pg_statio_user_tables`,
	).Scan(&stats.CacheHitRatio)
	if err != nil {
		return nil, fmt.Errorf("cache hit ratio: %w", err)
	}

	// Uptime
	var uptime string
	err = r.pool.QueryRow(ctx,
		`SELECT to_char(now() - pg_postmaster_start_time(), 'DD "d" HH24 "h" MI "m"')`,
	).Scan(&uptime)
	if err != nil {
		return nil, fmt.Errorf("uptime: %w", err)
	}
	stats.Uptime = uptime

	// Per-table sizes for application tables
	tables := []string{"users", "organizations", "boards", "board_versions", "board_assets", "audit_events", "memberships", "share_links", "accounts", "refresh_tokens", "backup_metadata", "backup_schedule"}
	for _, t := range tables {
		var info TableSizeInfo
		info.Table = t
		err = r.pool.QueryRow(ctx,
			`SELECT
				pg_total_relation_size($1::regclass),
				pg_table_size($1::regclass),
				pg_indexes_size($1::regclass),
				(SELECT COALESCE(n_live_tup, 0) FROM pg_stat_user_tables WHERE relname = $1::text)`,
			t,
		).Scan(&info.TotalBytes, &info.DataBytes, &info.IndexBytes, &info.RowCount)
		if err != nil {
			return nil, fmt.Errorf("table size %s: %w", t, err)
		}
		stats.Tables = append(stats.Tables, info)
	}

	return stats, nil
}

// GetSystemStats returns row counts, database metrics, pool stats, and runtime info.
func (r *AuditRepository) GetSystemStats(ctx context.Context) (*SystemStatsResult, error) {
	counts, err := r.CountAll(ctx)
	if err != nil {
		return nil, err
	}
	dbStats, err := r.GetDatabaseStats(ctx)
	if err != nil {
		return nil, err
	}

	// pgxpool connection pool stats
	ps := r.pool.Stat()
	poolStats := PoolStats{
		AcquireCount:         ps.AcquireCount(),
		AcquiredConns:        ps.AcquiredConns(),
		IdleConns:            ps.IdleConns(),
		TotalConns:           ps.TotalConns(),
		ConstructingConns:    ps.ConstructingConns(),
		MaxConns:             ps.MaxConns(),
		EmptyAcquireCount:    ps.EmptyAcquireCount(),
		CanceledAcquireCount: ps.CanceledAcquireCount(),
	}

	// Go runtime stats
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	runtimeStats := RuntimeStats{
		Goroutines: goruntime.NumGoroutine(),
		HeapAlloc:  m.HeapAlloc,
		HeapSys:    m.HeapSys,
		HeapInuse:  m.HeapInuse,
		StackInuse: m.StackInuse,
		GCPauseNs:  m.PauseNs[(m.NumGC+255)%256],
		NumGC:      m.NumGC,
		NumCPU:     goruntime.NumCPU(),
		GoVersion:  goruntime.Version(),
	}

	return &SystemStatsResult{
		Counts:   *counts,
		Database: *dbStats,
		Pool:     poolStats,
		Runtime:  runtimeStats,
	}, nil
}

// PurgeOlderThan deletes audit events older than the given cutoff date for an org.
// Returns the number of deleted rows.
func (r *AuditRepository) PurgeOlderThan(ctx context.Context, orgID string, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM audit_events WHERE org_id = $1 AND created_at < $2`,
		orgID, cutoff,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
