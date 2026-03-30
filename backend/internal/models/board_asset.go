package models

import "time"

// BoardAsset tracks files stored in S3/MinIO for a board.
type BoardAsset struct {
	ID         string    `json:"id"`
	BoardID    string    `json:"boardId"`
	FileID     string    `json:"fileId"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int64     `json:"sizeBytes"`
	StorageKey string    `json:"storageKey"`
	SHA256     string    `json:"sha256"`
	CreatedAt  time.Time `json:"createdAt"`
}
