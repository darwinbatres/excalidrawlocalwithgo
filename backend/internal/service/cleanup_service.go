package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/storage"
)

// CleanupService handles periodic and on-demand cleanup operations across the system.
// Designed as a reusable component that coordinates cleanup of orphaned assets,
// stale audit logs, and other maintenance tasks.
type CleanupService struct {
	assets   repository.BoardAssetRepo
	boards   repository.BoardRepo
	versions repository.BoardVersionRepo
	audit    repository.AuditRepo
	s3       storage.ObjectStorage
	log      zerolog.Logger
}

// NewCleanupService creates a CleanupService.
func NewCleanupService(
	assets repository.BoardAssetRepo,
	boards repository.BoardRepo,
	versions repository.BoardVersionRepo,
	audit repository.AuditRepo,
	s3 storage.ObjectStorage,
	log zerolog.Logger,
) *CleanupService {
	return &CleanupService{
		assets:   assets,
		boards:   boards,
		versions: versions,
		audit:    audit,
		s3:       s3,
		log:      log,
	}
}

// CleanupResult summarizes what a cleanup run achieved.
type CleanupResult struct {
	OrphanedAssetsRemoved int   `json:"orphanedAssetsRemoved"`
	AuditEventsPurged     int64 `json:"auditEventsPurged"`
	Errors                int   `json:"errors"`
}

// CleanupBoardAssets removes assets from S3 and DB that are no longer referenced
// by the board's current scene. Reads the latest version to determine active file IDs.
func (s *CleanupService) CleanupBoardAssets(ctx context.Context, boardID string) (int, error) {
	// Get the board's current version to find active file IDs
	bv, err := s.boards.GetByIDWithVersion(ctx, boardID)
	if err != nil {
		return 0, fmt.Errorf("getting board for cleanup: %w", err)
	}

	activeFileIDs := extractActiveFileIDs(bv)

	orphaned, err := s.assets.DeleteOrphaned(ctx, boardID, activeFileIDs)
	if err != nil {
		return 0, fmt.Errorf("deleting orphaned assets: %w", err)
	}

	// Best-effort S3 cleanup
	for _, asset := range orphaned {
		if delErr := s.s3.Delete(ctx, asset.StorageKey); delErr != nil {
			s.log.Warn().Err(delErr).Str("key", asset.StorageKey).Msg("failed to delete orphaned file from S3")
		}
	}

	if len(orphaned) > 0 {
		s.log.Info().Int("count", len(orphaned)).Str("boardID", boardID).Msg("cleaned orphaned board assets")
	}

	return len(orphaned), nil
}

// PurgeAuditLogs removes audit events older than the retention period for an org.
func (s *CleanupService) PurgeAuditLogs(ctx context.Context, orgID string, retentionDays int) (int64, error) {
	if retentionDays < 30 {
		retentionDays = 30
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.audit.PurgeOlderThan(ctx, orgID, cutoff)
	if err != nil {
		return 0, fmt.Errorf("purging audit logs: %w", err)
	}

	if deleted > 0 {
		s.log.Info().Int64("deleted", deleted).Str("orgID", orgID).Msg("purged old audit logs")
	}

	return deleted, nil
}

// extractActiveFileIDs pulls file IDs from a board's current scene that are referenced by image elements.
func extractActiveFileIDs(bv *models.BoardWithVersion) []string {
	if bv == nil {
		return nil
	}

	// Try to get scene JSON from the latest version
	var sceneJSON json.RawMessage
	if bv.LatestVersion != nil {
		sceneJSON = bv.LatestVersion.SceneJSON
	}

	if len(sceneJSON) == 0 {
		return nil
	}

	type elementRef struct {
		Type   string `json:"type"`
		FileID string `json:"fileId,omitempty"`
	}
	var scene struct {
		Elements []elementRef `json:"elements"`
	}
	if err := json.Unmarshal(sceneJSON, &scene); err != nil {
		return nil
	}

	var ids []string
	seen := make(map[string]struct{})
	for _, el := range scene.Elements {
		if el.Type == "image" && el.FileID != "" {
			if _, ok := seen[el.FileID]; !ok {
				seen[el.FileID] = struct{}{}
				ids = append(ids, el.FileID)
			}
		}
	}
	return ids
}
