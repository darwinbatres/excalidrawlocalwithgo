package service

import (
	"encoding/json"
	"testing"

	"github.com/darwinbatres/drawgo/backend/internal/models"
)

func TestExtractActiveFileIDs_NilBoardVersion(t *testing.T) {
	ids := extractActiveFileIDs(nil)
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
}

func TestExtractActiveFileIDs_NilLatestVersion(t *testing.T) {
	bv := &models.BoardWithVersion{}
	ids := extractActiveFileIDs(bv)
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
}

func TestExtractActiveFileIDs_EmptyScene(t *testing.T) {
	bv := &models.BoardWithVersion{
		LatestVersion: &models.BoardVersion{
			SceneJSON: json.RawMessage(`{}`),
		},
	}
	ids := extractActiveFileIDs(bv)
	if len(ids) != 0 {
		t.Errorf("expected 0 ids, got %d", len(ids))
	}
}

func TestExtractActiveFileIDs_ExtractsImageFileIDs(t *testing.T) {
	scene := `{
		"elements": [
			{"type": "image", "fileId": "abc123"},
			{"type": "text", "text": "hello"},
			{"type": "image", "fileId": "def456"},
			{"type": "rectangle"}
		]
	}`
	bv := &models.BoardWithVersion{
		LatestVersion: &models.BoardVersion{
			SceneJSON: json.RawMessage(scene),
		},
	}

	ids := extractActiveFileIDs(bv)
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d: %v", len(ids), ids)
	}

	expected := map[string]bool{"abc123": true, "def456": true}
	for _, id := range ids {
		if !expected[id] {
			t.Errorf("unexpected id: %s", id)
		}
	}
}

func TestExtractActiveFileIDs_DeduplicatesFileIDs(t *testing.T) {
	scene := `{
		"elements": [
			{"type": "image", "fileId": "same123"},
			{"type": "image", "fileId": "same123"},
			{"type": "image", "fileId": "other456"}
		]
	}`
	bv := &models.BoardWithVersion{
		LatestVersion: &models.BoardVersion{
			SceneJSON: json.RawMessage(scene),
		},
	}

	ids := extractActiveFileIDs(bv)
	if len(ids) != 2 {
		t.Fatalf("expected 2 unique ids, got %d: %v", len(ids), ids)
	}
}

func TestExtractActiveFileIDs_SkipsEmptyFileIDs(t *testing.T) {
	scene := `{
		"elements": [
			{"type": "image", "fileId": ""},
			{"type": "image", "fileId": "valid123"}
		]
	}`
	bv := &models.BoardWithVersion{
		LatestVersion: &models.BoardVersion{
			SceneJSON: json.RawMessage(scene),
		},
	}

	ids := extractActiveFileIDs(bv)
	if len(ids) != 1 {
		t.Fatalf("expected 1 id, got %d: %v", len(ids), ids)
	}
	if ids[0] != "valid123" {
		t.Errorf("expected 'valid123', got '%s'", ids[0])
	}
}

func TestExtractActiveFileIDs_InvalidJSON(t *testing.T) {
	bv := &models.BoardWithVersion{
		LatestVersion: &models.BoardVersion{
			SceneJSON: json.RawMessage(`{invalid json`),
		},
	}

	ids := extractActiveFileIDs(bv)
	if ids != nil {
		t.Errorf("expected nil for invalid JSON, got %v", ids)
	}
}

func TestExtractActiveFileIDs_NoImages(t *testing.T) {
	scene := `{
		"elements": [
			{"type": "text", "text": "hello"},
			{"type": "rectangle"},
			{"type": "arrow"}
		]
	}`
	bv := &models.BoardWithVersion{
		LatestVersion: &models.BoardVersion{
			SceneJSON: json.RawMessage(scene),
		},
	}

	ids := extractActiveFileIDs(bv)
	if len(ids) != 0 {
		t.Errorf("expected 0 ids, got %d: %v", len(ids), ids)
	}
}
