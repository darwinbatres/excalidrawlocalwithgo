package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
	"github.com/darwinbatres/drawgo/backend/internal/storage"
)

// Allowed MIME types for image uploads.
var allowedMIMETypes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// FileService handles image storage, presigned URLs, and board asset tracking.
type FileService struct {
	assets      repository.BoardAssetRepo
	boards      repository.BoardRepo
	versions    repository.BoardVersionRepo
	s3          storage.ObjectStorage
	access      *AccessService
	audit       repository.AuditRepo
	log         zerolog.Logger
	maxFileSize int64
}

// NewFileService creates a FileService.
func NewFileService(
	assets repository.BoardAssetRepo,
	boards repository.BoardRepo,
	versions repository.BoardVersionRepo,
	s3 storage.ObjectStorage,
	access *AccessService,
	audit repository.AuditRepo,
	log zerolog.Logger,
	maxFileSize int64,
) *FileService {
	return &FileService{
		assets:      assets,
		boards:      boards,
		versions:    versions,
		s3:          s3,
		access:      access,
		audit:       audit,
		log:         log,
		maxFileSize: maxFileSize,
	}
}

// UploadResult is the response for a successful file upload.
type UploadResult struct {
	FileID     string `json:"fileId"`
	StorageKey string `json:"storageKey"`
	MimeType   string `json:"mimeType"`
	SizeBytes  int64  `json:"sizeBytes"`
	SHA256     string `json:"sha256"`
	URL        string `json:"url"`
}

// UploadFile stores a file in S3 and tracks it as a board asset.
func (s *FileService) UploadFile(ctx context.Context, userID, boardID, fileID string, data io.Reader, size int64, contentType string) (*UploadResult, *apierror.Error) {
	// Verify edit access
	if _, apiErr := s.access.RequireBoardEdit(ctx, userID, boardID); apiErr != nil {
		return nil, apiErr
	}

	// Validate MIME type
	if !allowedMIMETypes[contentType] {
		return nil, apierror.ErrBadRequest.WithMessage(
			fmt.Sprintf("Unsupported file type: %s. Allowed: png, jpeg, gif, webp, svg+xml", contentType))
	}

	// Validate size
	if size > s.maxFileSize {
		return nil, apierror.ErrRequestTooLarge.WithMessage(
			fmt.Sprintf("File too large: %d bytes (max %d)", size, s.maxFileSize))
	}

	// Read body into memory for hashing + upload
	buf, err := io.ReadAll(io.LimitReader(data, s.maxFileSize+1))
	if err != nil {
		s.log.Error().Err(err).Msg("failed to read upload body")
		return nil, apierror.ErrInternal
	}
	if int64(len(buf)) > s.maxFileSize {
		return nil, apierror.ErrRequestTooLarge
	}

	// Validate magic bytes to confirm MIME type matches content
	detected := http.DetectContentType(buf)
	if !isContentTypeCompatible(contentType, detected) {
		return nil, apierror.ErrBadRequest.WithMessage("File content does not match declared content type")
	}

	// Compute SHA256
	hash := sha256.Sum256(buf)
	hashHex := hex.EncodeToString(hash[:])

	// Check for dedup: if same hash exists for this board, reuse storage key
	existing, err := s.assets.FindBySHA256(ctx, boardID, hashHex)
	if err == nil && existing != nil && existing.FileID == fileID {
		// Same file, same hash — no-op, return existing
		url, _ := s.s3.PresignedURL(ctx, existing.StorageKey)
		return &UploadResult{
			FileID:     existing.FileID,
			StorageKey: existing.StorageKey,
			MimeType:   existing.MimeType,
			SizeBytes:  existing.SizeBytes,
			SHA256:     existing.SHA256,
			URL:        url,
		}, nil
	}

	// Build storage key: boards/{boardID}/files/{fileID}
	storageKey := fmt.Sprintf("boards/%s/files/%s", boardID, fileID)

	// Upload to S3
	if err := s.s3.Upload(ctx, storageKey, bytes.NewReader(buf), int64(len(buf)), contentType); err != nil {
		s.log.Error().Err(err).Str("key", storageKey).Msg("failed to upload file to S3")
		return nil, apierror.ErrInternal.WithMessage("Failed to store file")
	}

	// Track asset in DB
	asset := &models.BoardAsset{
		BoardID:    boardID,
		FileID:     fileID,
		MimeType:   contentType,
		SizeBytes:  int64(len(buf)),
		StorageKey: storageKey,
		SHA256:     hashHex,
	}
	if err := s.assets.Upsert(ctx, asset); err != nil {
		s.log.Error().Err(err).Str("key", storageKey).Msg("failed to track asset in DB")
		// Best effort: file is in S3 but not tracked — log and return error
		return nil, apierror.ErrInternal
	}

	// Generate presigned URL for immediate use
	url, _ := s.s3.PresignedURL(ctx, storageKey)

	s.logAuditAsync(userID, boardID, models.AuditActionFileUpload, "board_asset", asset.ID)

	return &UploadResult{
		FileID:     fileID,
		StorageKey: storageKey,
		MimeType:   contentType,
		SizeBytes:  int64(len(buf)),
		SHA256:     hashHex,
		URL:        url,
	}, nil
}

// GetPresignedURL generates a presigned download URL for a board's file.
func (s *FileService) GetPresignedURL(ctx context.Context, userID, boardID, fileID string) (string, *apierror.Error) {
	// Verify view access
	if _, apiErr := s.access.RequireBoardView(ctx, userID, boardID); apiErr != nil {
		return "", apiErr
	}

	asset, err := s.assets.GetByBoardAndFileID(ctx, boardID, fileID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", apierror.ErrNotFound.WithMessage("File not found")
		}
		s.log.Error().Err(err).Msg("failed to look up board asset")
		return "", apierror.ErrInternal
	}

	url, err := s.s3.PresignedURL(ctx, asset.StorageKey)
	if err != nil {
		s.log.Error().Err(err).Str("key", asset.StorageKey).Msg("failed to generate presigned URL")
		return "", apierror.ErrInternal
	}

	return url, nil
}

// DownloadFile streams a file directly from S3.
func (s *FileService) DownloadFile(ctx context.Context, userID, boardID, fileID string) (io.ReadCloser, string, *apierror.Error) {
	if _, apiErr := s.access.RequireBoardView(ctx, userID, boardID); apiErr != nil {
		return nil, "", apiErr
	}

	asset, err := s.assets.GetByBoardAndFileID(ctx, boardID, fileID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, "", apierror.ErrNotFound.WithMessage("File not found")
		}
		s.log.Error().Err(err).Msg("failed to look up board asset")
		return nil, "", apierror.ErrInternal
	}

	reader, err := s.s3.Download(ctx, asset.StorageKey)
	if err != nil {
		s.log.Error().Err(err).Str("key", asset.StorageKey).Msg("failed to download from S3")
		return nil, "", apierror.ErrInternal
	}

	return reader, asset.MimeType, nil
}

// ListAssets returns all tracked assets for a board.
func (s *FileService) ListAssets(ctx context.Context, userID, boardID string) ([]models.BoardAsset, *apierror.Error) {
	if _, apiErr := s.access.RequireBoardView(ctx, userID, boardID); apiErr != nil {
		return nil, apiErr
	}

	assets, err := s.assets.ListByBoard(ctx, boardID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list board assets")
		return nil, apierror.ErrInternal
	}

	return assets, nil
}

// BoardStorageInfo is the storage summary for a single board.
type BoardStorageInfo struct {
	BoardID        string `json:"boardId"`
	TotalBytes     int64  `json:"totalBytes"`
	AssetBytes     int64  `json:"assetBytes"`
	SceneBytes     int64  `json:"sceneBytes"`
	AppStateBytes  int64  `json:"appStateBytes"`
	ThumbnailBytes int64  `json:"thumbnailBytes"`
	AssetCount     int    `json:"assetCount"`
	VersionCount   int64  `json:"versionCount"`
}

// GetBoardStorage calculates storage used by a board's assets and data.
func (s *FileService) GetBoardStorage(ctx context.Context, userID, boardID string) (*BoardStorageInfo, *apierror.Error) {
	if _, apiErr := s.access.RequireBoardView(ctx, userID, boardID); apiErr != nil {
		return nil, apiErr
	}

	assetTotal, err := s.assets.StorageUsedByBoard(ctx, boardID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to calculate board asset storage")
		return nil, apierror.ErrInternal
	}

	assets, err := s.assets.ListByBoard(ctx, boardID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list board assets for count")
		return nil, apierror.ErrInternal
	}

	dataInfo, err := s.versions.DataStorageByBoard(ctx, boardID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to calculate board data storage")
		return nil, apierror.ErrInternal
	}

	return &BoardStorageInfo{
		BoardID:        boardID,
		TotalBytes:     assetTotal + dataInfo.SceneBytes + dataInfo.AppStateBytes + dataInfo.ThumbnailBytes,
		AssetBytes:     assetTotal,
		SceneBytes:     dataInfo.SceneBytes,
		AppStateBytes:  dataInfo.AppStateBytes,
		ThumbnailBytes: dataInfo.ThumbnailBytes,
		AssetCount:     len(assets),
		VersionCount:   dataInfo.VersionCount,
	}, nil
}

// OrgStorageInfo is the storage summary for an organization.
type OrgStorageInfo struct {
	OrgID          string `json:"orgId"`
	TotalBytes     int64  `json:"totalBytes"`
	AssetBytes     int64  `json:"assetBytes"`
	SceneBytes     int64  `json:"sceneBytes"`
	AppStateBytes  int64  `json:"appStateBytes"`
	ThumbnailBytes int64  `json:"thumbnailBytes"`
}

// GetOrgStorage calculates total storage used by all boards in an org.
func (s *FileService) GetOrgStorage(ctx context.Context, userID, orgID string) (*OrgStorageInfo, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleViewer); apiErr != nil {
		return nil, apiErr
	}

	assetTotal, err := s.assets.StorageUsedByOrg(ctx, orgID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to calculate org asset storage")
		return nil, apierror.ErrInternal
	}

	dataInfo, err := s.versions.DataStorageByOrg(ctx, orgID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to calculate org data storage")
		return nil, apierror.ErrInternal
	}

	return &OrgStorageInfo{
		OrgID:          orgID,
		TotalBytes:     assetTotal + dataInfo.SceneBytes + dataInfo.AppStateBytes + dataInfo.ThumbnailBytes,
		AssetBytes:     assetTotal,
		SceneBytes:     dataInfo.SceneBytes,
		AppStateBytes:  dataInfo.AppStateBytes,
		ThumbnailBytes: dataInfo.ThumbnailBytes,
	}, nil
}

// UploadThumbnail decodes a base64 data URL and uploads it to S3 as the board's thumbnail.
func (s *FileService) UploadThumbnail(ctx context.Context, userID, boardID, dataURL string) (string, *apierror.Error) {
	if _, apiErr := s.access.RequireBoardEdit(ctx, userID, boardID); apiErr != nil {
		return "", apiErr
	}

	// Parse data URL: data:<mime>;base64,<data>
	mimeType, rawData, err := parseDataURL(dataURL)
	if err != nil {
		return "", apierror.ErrBadRequest.WithMessage("Invalid thumbnail data URL")
	}

	if !allowedMIMETypes[mimeType] {
		return "", apierror.ErrBadRequest.WithMessage("Unsupported thumbnail type")
	}

	storageKey := fmt.Sprintf("boards/%s/thumbnail", boardID)

	if err := s.s3.Upload(ctx, storageKey, bytes.NewReader(rawData), int64(len(rawData)), mimeType); err != nil {
		s.log.Error().Err(err).Str("boardID", boardID).Msg("failed to upload thumbnail")
		return "", apierror.ErrInternal
	}

	url, err := s.s3.PresignedURL(ctx, storageKey)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to generate thumbnail URL")
		return "", apierror.ErrInternal
	}

	return url, nil
}

// CleanOrphanedAssets identifies files tracked in DB but not referenced by the scene's image elements,
// deletes them from both S3 and the DB.
func (s *FileService) CleanOrphanedAssets(ctx context.Context, boardID string, sceneJSON json.RawMessage) (int, error) {
	activeFileIDs := extractImageFileIDs(sceneJSON)

	orphaned, err := s.assets.DeleteOrphaned(ctx, boardID, activeFileIDs)
	if err != nil {
		return 0, fmt.Errorf("deleting orphaned assets from DB: %w", err)
	}

	// Best-effort cleanup from S3
	for _, asset := range orphaned {
		if err := s.s3.Delete(ctx, asset.StorageKey); err != nil {
			s.log.Warn().Err(err).Str("key", asset.StorageKey).Msg("failed to delete orphaned file from S3")
		}
	}

	if len(orphaned) > 0 {
		s.log.Info().Int("count", len(orphaned)).Str("boardID", boardID).Msg("cleaned orphaned assets")
	}

	return len(orphaned), nil
}

// ExtractAndUploadImages extracts base64 files from scene JSON and uploads them to S3,
// returning an updated scene with files removed from the inline JSON.
func (s *FileService) ExtractAndUploadImages(ctx context.Context, boardID string, sceneJSON json.RawMessage) (json.RawMessage, int, error) {
	if len(sceneJSON) == 0 {
		return sceneJSON, 0, nil
	}

	// Partial unmarshal to keep unknown fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(sceneJSON, &raw); err != nil {
		return sceneJSON, 0, nil
	}

	var files map[string]sceneFile
	if filesRaw, ok := raw["files"]; ok {
		if err := json.Unmarshal(filesRaw, &files); err != nil {
			return sceneJSON, 0, nil
		}
	}

	if len(files) == 0 {
		return sceneJSON, 0, nil
	}

	uploaded := 0
	for fileID, file := range files {
		if file.DataURL == "" {
			continue
		}

		mimeType, data, err := parseDataURL(file.DataURL)
		if err != nil {
			s.log.Warn().Str("fileID", fileID).Msg("skipping file with invalid data URL")
			continue
		}

		if !allowedMIMETypes[mimeType] {
			continue
		}

		storageKey := fmt.Sprintf("boards/%s/files/%s", boardID, fileID)
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		if err := s.s3.Upload(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
			s.log.Error().Err(err).Str("fileID", fileID).Msg("failed to upload extracted image")
			continue
		}

		asset := &models.BoardAsset{
			BoardID:    boardID,
			FileID:     fileID,
			MimeType:   mimeType,
			SizeBytes:  int64(len(data)),
			StorageKey: storageKey,
			SHA256:     hashHex,
		}
		if err := s.assets.Upsert(ctx, asset); err != nil {
			s.log.Error().Err(err).Str("fileID", fileID).Msg("failed to track extracted image asset")
			continue
		}

		uploaded++
	}

	// Remove files from scene JSON to reduce storage
	if uploaded > 0 {
		delete(raw, "files")
		cleaned, err := json.Marshal(raw)
		if err != nil {
			return sceneJSON, uploaded, nil
		}
		return cleaned, uploaded, nil
	}

	return sceneJSON, 0, nil
}

// MirrorImagesToS3 extracts base64 images from scene JSON and uploads them to S3
// for metrics and backup, WITHOUT modifying the scene JSON. This allows S3 storage
// metrics to reflect actual image usage while keeping images inline for Excalidraw.
func (s *FileService) MirrorImagesToS3(ctx context.Context, boardID string, sceneJSON json.RawMessage) {
	if len(sceneJSON) == 0 {
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(sceneJSON, &raw); err != nil {
		return
	}

	var files map[string]sceneFile
	if filesRaw, ok := raw["files"]; ok {
		if err := json.Unmarshal(filesRaw, &files); err != nil {
			return
		}
	}

	if len(files) == 0 {
		return
	}

	uploaded := 0
	for fileID, file := range files {
		if file.DataURL == "" {
			continue
		}

		mimeType, data, err := parseDataURL(file.DataURL)
		if err != nil {
			continue
		}

		if !allowedMIMETypes[mimeType] {
			continue
		}

		storageKey := fmt.Sprintf("boards/%s/files/%s", boardID, fileID)
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		if err := s.s3.Upload(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
			s.log.Error().Err(err).Str("fileID", fileID).Msg("failed to mirror image to S3")
			continue
		}

		asset := &models.BoardAsset{
			BoardID:    boardID,
			FileID:     fileID,
			MimeType:   mimeType,
			SizeBytes:  int64(len(data)),
			StorageKey: storageKey,
			SHA256:     hashHex,
		}
		if err := s.assets.Upsert(ctx, asset); err != nil {
			s.log.Error().Err(err).Str("fileID", fileID).Msg("failed to track mirrored image asset")
			continue
		}

		uploaded++
	}

	if uploaded > 0 {
		s.log.Info().Int("count", uploaded).Str("boardId", boardID).Msg("mirrored images to S3")
	}
}

func (s *FileService) logAuditAsync(actorID, boardID, action, targetType, targetID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.audit.Log(ctx, "", &actorID, action, targetType, targetID, nil, nil, nil); err != nil {
			s.log.Error().Err(err).Str("action", action).Msg("failed to log audit event")
		}
	}()
}

// sceneFile represents a file entry in an Excalidraw scene.
type sceneFile struct {
	MimeType string `json:"mimeType,omitempty"`
	DataURL  string `json:"dataURL,omitempty"`
}

// extractImageFileIDs collects all file IDs referenced by image elements in the scene.
func extractImageFileIDs(sceneJSON json.RawMessage) []string {
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

// parseDataURL parses a data URL (data:<mime>;base64,<data>) into MIME type and decoded bytes.
func parseDataURL(dataURL string) (string, []byte, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("not a data URL")
	}

	// data:<mime>;base64,<data>
	rest := dataURL[5:] // strip "data:"
	semicolonIdx := strings.Index(rest, ";")
	if semicolonIdx < 0 {
		return "", nil, fmt.Errorf("missing semicolon in data URL")
	}

	mimeType := rest[:semicolonIdx]
	afterSemicolon := rest[semicolonIdx+1:]

	if !strings.HasPrefix(afterSemicolon, "base64,") {
		return "", nil, fmt.Errorf("not base64 encoded")
	}

	encoded := afterSemicolon[7:] // strip "base64,"
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// Try with RawStdEncoding (no padding)
		data, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return "", nil, fmt.Errorf("invalid base64: %w", err)
		}
	}

	return mimeType, data, nil
}

// isContentTypeCompatible checks if the declared MIME type is compatible with the detected type.
func isContentTypeCompatible(declared, detected string) bool {
	// http.DetectContentType returns generic types; be lenient for images
	if strings.HasPrefix(declared, "image/") {
		// DetectContentType may return "image/png", "image/jpeg", "image/gif", "image/webp"
		// or "application/octet-stream" / "text/xml" for SVG
		if detected == "application/octet-stream" || detected == "text/xml; charset=utf-8" || detected == "text/plain; charset=utf-8" {
			return true // Accept — these are common for SVG or small images
		}
		if strings.HasPrefix(detected, "image/") {
			return true
		}
	}
	return false
}
