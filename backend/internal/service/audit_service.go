package service

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/buildinfo"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/logbuffer"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/storage"
)

// processStartTime records when the process started.
var processStartTime = time.Now()

// AuditService handles audit log queries, statistics, and cleanup operations.
type AuditService struct {
	audit          repository.AuditRepo
	assets         repository.BoardAssetRepo
	backup         repository.BackupRepo
	access         *AccessService
	s3             storage.ObjectStorage
	log            zerolog.Logger
	s3Bucket       string
	backupBucket   string
	requestMetrics *middleware.RequestMetrics
	bruteForce     *middleware.BruteForce
	logBuffer      *logbuffer.RingBuffer
}

// NewAuditService creates an AuditService.
func NewAuditService(
	audit repository.AuditRepo,
	assets repository.BoardAssetRepo,
	backup repository.BackupRepo,
	access *AccessService,
	s3 storage.ObjectStorage,
	log zerolog.Logger,
	s3Bucket string,
	backupBucket string,
) *AuditService {
	return &AuditService{
		audit:        audit,
		assets:       assets,
		backup:       backup,
		access:       access,
		s3:           s3,
		log:          log,
		s3Bucket:     s3Bucket,
		backupBucket: backupBucket,
	}
}

// SetRequestMetrics attaches the request metrics collector to the service.
func (s *AuditService) SetRequestMetrics(rm *middleware.RequestMetrics) {
	s.requestMetrics = rm
}

// SetBruteForce attaches the brute-force protector for stats reporting.
func (s *AuditService) SetBruteForce(bf *middleware.BruteForce) {
	s.bruteForce = bf
}

// SetLogBuffer attaches the log ring buffer for log summary stats.
func (s *AuditService) SetLogBuffer(lb *logbuffer.RingBuffer) {
	s.logBuffer = lb
}

// ListAuditLogs returns filtered audit events for an org. Requires ADMIN+.
func (s *AuditService) ListAuditLogs(ctx context.Context, userID, orgID string, params repository.AuditQueryParams) (*repository.AuditQueryResult, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleAdmin); apiErr != nil {
		return nil, apiErr
	}

	params.OrgID = orgID
	result, err := s.audit.Query(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to query audit logs")
		return nil, apierror.ErrInternal
	}

	return result, nil
}

// GetAuditStats returns aggregated audit statistics for an org. Requires ADMIN+.
func (s *AuditService) GetAuditStats(ctx context.Context, userID, orgID string, days int) (*repository.AuditStats, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleAdmin); apiErr != nil {
		return nil, apiErr
	}

	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	since := time.Now().AddDate(0, 0, -days)
	stats, err := s.audit.Stats(ctx, orgID, since)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get audit stats")
		return nil, apierror.ErrInternal
	}

	return stats, nil
}

// OrgStats returns counts and CRUD breakdown scoped to a specific organization. Requires VIEWER+.
func (s *AuditService) OrgStats(ctx context.Context, userID, orgID string) (*repository.OrgStatsResult, *apierror.Error) {
	_, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleViewer)
	if apiErr != nil {
		return nil, apiErr
	}

	orgName, counts, err := s.audit.CountAllByOrg(ctx, orgID)
	if err != nil {
		s.log.Error().Err(err).Str("orgID", orgID).Msg("failed to get org counts")
		return nil, apierror.ErrInternal
	}

	result := &repository.OrgStatsResult{
		OrgID:   orgID,
		OrgName: orgName,
		Counts:  *counts,
	}

	// CRUD breakdown for this org
	crud, crudErr := s.audit.OrgCRUDBreakdown(ctx, orgID)
	if crudErr != nil {
		s.log.Warn().Err(crudErr).Str("orgID", orgID).Msg("failed to get org CRUD breakdown")
	} else {
		result.CRUDBreakdown = crud
	}

	return result, nil
}

// SystemStats returns system-wide table counts, database metrics, S3 storage, process stats, and build info.
func (s *AuditService) SystemStats(ctx context.Context) (*repository.SystemStatsResult, *apierror.Error) {
	result, err := s.audit.GetSystemStats(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get system stats")
		return nil, apierror.ErrInternal
	}

	// Collect S3 storage stats for all buckets
	storageStats := repository.S3StorageStats{}
	buckets := []string{s.s3Bucket}
	if s.backupBucket != "" && s.backupBucket != s.s3Bucket {
		buckets = append(buckets, s.backupBucket)
	}
	for _, bucket := range buckets {
		bs, bErr := s.s3.BucketStats(ctx, bucket)
		if bErr != nil {
			s.log.Warn().Err(bErr).Str("bucket", bucket).Msg("failed to get bucket stats")
			continue
		}
		info := repository.BucketInfo{
			Bucket:       bs.Bucket,
			ObjectCount:  bs.ObjectCount,
			TotalBytes:   bs.TotalBytes,
			LargestBytes: bs.LargestBytes,
		}
		storageStats.Buckets = append(storageStats.Buckets, info)
		storageStats.TotalBytes += bs.TotalBytes
		storageStats.TotalObjects += bs.ObjectCount
	}
	result.Storage = storageStats

	// Process metrics
	result.Process = getProcessStats()

	// Container metrics (cgroup-aware)
	result.Container = getContainerStats()

	// Request metrics
	if s.requestMetrics != nil {
		result.Requests = s.requestMetrics.Snapshot()
	}

	// Brute-force protection stats
	if s.bruteForce != nil {
		result.BruteForce = s.bruteForce.Stats()
	}

	// Build info
	result.Build = repository.BuildInfo{
		Version:   buildinfo.Version,
		CommitSHA: buildinfo.CommitSHA,
		BuildTime: buildinfo.BuildTime,
		GoVersion: goruntime.Version(),
	}

	// CRUD operation breakdown (all audit events grouped by action)
	crud, crudErr := s.audit.GlobalCRUDBreakdown(ctx)
	if crudErr != nil {
		s.log.Warn().Err(crudErr).Msg("failed to get global CRUD breakdown")
	} else {
		result.CRUDBreakdown = crud
	}

	// Backup stats
	bkStats := s.getBackupStats(ctx)
	result.BackupInfo = bkStats

	// Log summary from ring buffer
	if s.logBuffer != nil {
		summary := s.logBuffer.Summary()
		result.Logs = &repository.LogSummary{
			Debug: summary.Debug,
			Info:  summary.Info,
			Warn:  summary.Warn,
			Error: summary.Error,
			Fatal: summary.Fatal,
			Total: summary.Total,
		}
	}

	return result, nil
}

// getBackupStats collects backup-related metrics from the backup repo.
func (s *AuditService) getBackupStats(ctx context.Context) *repository.BackupStats {
	if s.backup == nil {
		return nil
	}

	stats := &repository.BackupStats{}

	// Get last backup info
	backups, total, err := s.backup.ListMetadata(ctx, 1, 0)
	if err != nil {
		s.log.Warn().Err(err).Msg("failed to list backups for stats")
		return stats
	}
	stats.TotalBackups = total

	if len(backups) > 0 {
		last := backups[0]
		stats.LastBackupAt = last.CreatedAt.UTC().Format(time.RFC3339)
		stats.LastBackupSize = last.SizeBytes
		stats.LastBackupStatus = last.Status
	}

	// Get schedule
	schedule, err := s.backup.GetSchedule(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("failed to get backup schedule for stats")
		return stats
	}
	stats.ScheduleEnabled = schedule.Enabled
	stats.ScheduleCron = schedule.CronExpr

	return stats
}

// getProcessStats collects OS process-level metrics.
func getProcessStats() repository.ProcessStats {
	pid := os.Getpid()
	uptime := int64(time.Since(processStartTime).Seconds())

	// Read RSS from /proc/self/statm (Linux)
	var rss uint64
	if data, err := os.ReadFile("/proc/self/statm"); err == nil {
		var fields [7]uint64
		n, _ := fmt.Sscan(string(data), &fields[0], &fields[1], &fields[2], &fields[3], &fields[4], &fields[5], &fields[6])
		if n >= 2 {
			rss = fields[1] * 4096 // pages → bytes (4KB page size)
		}
	}

	// Count open file descriptors
	openFDs := 0
	if entries, err := os.ReadDir("/proc/self/fd"); err == nil {
		openFDs = len(entries)
	}

	return repository.ProcessStats{
		PID:           pid,
		RSS:           rss,
		OpenFDs:       openFDs,
		UptimeSeconds: uptime,
		StartTime:     processStartTime.UTC().Format(time.RFC3339),
	}
}

// getContainerStats reads cgroup v2 (then v1) metrics for container-aware stats.
func getContainerStats() repository.ContainerStats {
	cs := repository.ContainerStats{}

	// Detect container via /.dockerenv or /run/.containerenv
	_, dockerErr := os.Stat("/.dockerenv")
	_, containerErr := os.Stat("/run/.containerenv")
	cs.IsContainer = dockerErr == nil || containerErr == nil

	// Try cgroup v2 first, then v1
	memLimit := readCgroupInt64("/sys/fs/cgroup/memory.max")
	if memLimit <= 0 {
		memLimit = readCgroupInt64("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	}
	// Ignore unreasonably large limits (host machine, no limit set)
	if memLimit > 0 && memLimit < 1<<62 {
		cs.MemoryLimitBytes = memLimit
	}

	memUsage := readCgroupInt64("/sys/fs/cgroup/memory.current")
	if memUsage <= 0 {
		memUsage = readCgroupInt64("/sys/fs/cgroup/memory/memory.usage_in_bytes")
	}
	if memUsage > 0 {
		cs.MemoryUsageBytes = memUsage
	}

	if cs.MemoryLimitBytes > 0 && cs.MemoryUsageBytes > 0 {
		cs.MemoryUsagePercent = float64(cs.MemoryUsageBytes) / float64(cs.MemoryLimitBytes) * 100
	}

	// CPU quota / period (cgroup v2: cpu.max "quota period", cgroup v1: separate files)
	if data, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(data)))
		if len(parts) >= 2 && parts[0] != "max" {
			cs.CPUQuota, _ = strconv.ParseInt(parts[0], 10, 64)
			cs.CPUPeriod, _ = strconv.ParseInt(parts[1], 10, 64)
		}
	} else {
		cs.CPUQuota = readCgroupInt64("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
		cs.CPUPeriod = readCgroupInt64("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	}
	if cs.CPUQuota > 0 && cs.CPUPeriod > 0 {
		cs.EffectiveCPUs = float64(cs.CPUQuota) / float64(cs.CPUPeriod)
	}

	// Container ID from /proc/self/cgroup or hostname
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(line, "/", 3)
			if len(parts) == 3 {
				id := parts[len(parts)-1]
				id = strings.TrimSpace(id)
				if len(id) >= 12 {
					cs.ContainerID = id
					if len(cs.ContainerID) > 12 {
						cs.ContainerID = cs.ContainerID[:12]
					}
					break
				}
			}
		}
	}
	if cs.ContainerID == "" && cs.IsContainer {
		if hostname, err := os.Hostname(); err == nil {
			cs.ContainerID = hostname
		}
	}

	// CPU usage and throttling from cgroup cpu.stat
	if data, err := os.ReadFile("/sys/fs/cgroup/cpu.stat"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				switch parts[0] {
				case "usage_usec":
					cs.CPUUsageUsec = val
				case "throttled_usec":
					cs.ThrottledUsec = val
				case "nr_throttled":
					cs.ThrottledPeriods = val
				}
			}
		}
	}

	// Memory cache from cgroup memory.stat
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.stat"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				switch parts[0] {
				case "file":
					cs.MemoryCacheBytes = val
				}
			}
		}
	}

	// Swap usage
	cs.SwapUsageBytes = readCgroupInt64("/sys/fs/cgroup/memory.swap.current")

	// OOM kills from cgroup memory.events
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.events"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 && parts[0] == "oom_kill" {
				cs.OOMKills, _ = strconv.ParseInt(parts[1], 10, 64)
			}
		}
	}

	// PID limits
	cs.PIDsCurrent = readCgroupInt64("/sys/fs/cgroup/pids.current")
	pidsMax := readCgroupInt64("/sys/fs/cgroup/pids.max")
	if pidsMax > 0 && pidsMax < 1<<62 {
		cs.PIDsLimit = pidsMax
	}

	// Network I/O from /proc/self/net/dev
	if data, err := os.ReadFile("/proc/self/net/dev"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "eth") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				rxBytes, _ := strconv.ParseUint(fields[1], 10, 64)
				rxPackets, _ := strconv.ParseUint(fields[2], 10, 64)
				txBytes, _ := strconv.ParseUint(fields[9], 10, 64)
				txPackets, _ := strconv.ParseUint(fields[10], 10, 64)
				cs.NetworkRxBytes += rxBytes
				cs.NetworkTxBytes += txBytes
				cs.NetworkRxPackets += rxPackets
				cs.NetworkTxPackets += txPackets
			}
		}
	}

	return cs
}

func readCgroupInt64(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// PurgeAuditLogs deletes audit events older than the specified number of days for an org.
// Requires OWNER role.
func (s *AuditService) PurgeAuditLogs(ctx context.Context, userID, orgID string, olderThanDays int) (int64, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleOwner); apiErr != nil {
		return 0, apiErr
	}

	if olderThanDays < 30 {
		return 0, apierror.ErrBadRequest.WithMessage("Minimum retention period is 30 days")
	}

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	deleted, err := s.audit.PurgeOlderThan(ctx, orgID, cutoff)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to purge audit logs")
		return 0, apierror.ErrInternal
	}

	s.log.Info().Int64("deleted", deleted).Str("orgID", orgID).Int("olderThanDays", olderThanDays).Msg("purged audit logs")
	return deleted, nil
}

// CleanupBoardAssets removes orphaned assets for a board based on its current scene.
// Returns the number of cleaned assets. This delegates to FileService.CleanOrphanedAssets
// but is exposed here for cleanup orchestration.
func (s *AuditService) CleanupBoardAssets(ctx context.Context, boardID string, activeFileIDs []string) (int, error) {
	orphaned, err := s.assets.DeleteOrphaned(ctx, boardID, activeFileIDs)
	if err != nil {
		return 0, err
	}

	// Best-effort S3 cleanup
	for _, asset := range orphaned {
		if delErr := s.s3.Delete(ctx, asset.StorageKey); delErr != nil {
			s.log.Warn().Err(delErr).Str("key", asset.StorageKey).Msg("failed to delete orphaned file from S3")
		}
	}

	return len(orphaned), nil
}

// StorageSummary represents a combined storage summary for an org.
type StorageSummary struct {
	OrgID          string `json:"orgId"`
	TotalFileBytes int64  `json:"totalFileBytes"`
	BoardCount     int64  `json:"boardCount"`
}

// GetWorkspaceStorage returns total storage information for an org workspace. Requires VIEWER+.
func (s *AuditService) GetWorkspaceStorage(ctx context.Context, userID, orgID string) (*StorageSummary, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleViewer); apiErr != nil {
		return nil, apiErr
	}

	totalBytes, err := s.assets.StorageUsedByOrg(ctx, orgID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get workspace storage")
		return nil, apierror.ErrInternal
	}

	counts, err := s.audit.CountAll(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get board count")
		return nil, apierror.ErrInternal
	}

	return &StorageSummary{
		OrgID:          orgID,
		TotalFileBytes: totalBytes,
		BoardCount:     counts.Boards,
	}, nil
}
