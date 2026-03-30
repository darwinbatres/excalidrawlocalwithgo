package service

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseDataURL_Valid(t *testing.T) {
	data := []byte("hello world")
	encoded := base64.StdEncoding.EncodeToString(data)
	dataURL := "data:image/png;base64," + encoded

	mime, result, err := parseDataURL(dataURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected mime 'image/png', got '%s'", mime)
	}
	if string(result) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(result))
	}
}

func TestParseDataURL_NoPadding(t *testing.T) {
	// base64 without padding
	data := []byte("hi")
	encoded := base64.RawStdEncoding.EncodeToString(data)
	dataURL := "data:image/jpeg;base64," + encoded

	mime, result, err := parseDataURL(dataURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("expected mime 'image/jpeg', got '%s'", mime)
	}
	if string(result) != "hi" {
		t.Errorf("expected 'hi', got '%s'", string(result))
	}
}

func TestParseDataURL_InvalidPrefix(t *testing.T) {
	_, _, err := parseDataURL("not-a-data-url")
	if err == nil {
		t.Fatal("expected error for non-data URL")
	}
}

func TestParseDataURL_MissingSemicolon(t *testing.T) {
	_, _, err := parseDataURL("data:image/pngbase64,abc")
	if err == nil {
		t.Fatal("expected error for missing semicolon")
	}
}

func TestParseDataURL_NotBase64(t *testing.T) {
	_, _, err := parseDataURL("data:image/png;charset=utf-8,hello")
	if err == nil {
		t.Fatal("expected error for non-base64 encoding")
	}
}

func TestIsContentTypeCompatible_ImageMatch(t *testing.T) {
	tests := []struct {
		declared string
		detected string
		want     bool
	}{
		{"image/png", "image/png", true},
		{"image/jpeg", "image/jpeg", true},
		{"image/svg+xml", "text/xml; charset=utf-8", true},
		{"image/svg+xml", "text/plain; charset=utf-8", true},
		{"image/png", "application/octet-stream", true},
		{"image/gif", "image/gif", true},
		{"image/webp", "image/webp", true},
		{"text/html", "text/html", false},
	}

	for _, tt := range tests {
		t.Run(tt.declared+"_"+tt.detected, func(t *testing.T) {
			got := isContentTypeCompatible(tt.declared, tt.detected)
			if got != tt.want {
				t.Errorf("isContentTypeCompatible(%q, %q) = %v, want %v", tt.declared, tt.detected, got, tt.want)
			}
		})
	}
}

func TestExtractImageFileIDs_NoElements(t *testing.T) {
	result := extractImageFileIDs(json.RawMessage(`{"elements":[]}`))
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestExtractImageFileIDs_NilScene(t *testing.T) {
	result := extractImageFileIDs(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestExtractImageFileIDs_MixedElements(t *testing.T) {
	scene := `{"elements":[
		{"type":"text","text":"hello"},
		{"type":"image","fileId":"abc123"},
		{"type":"rectangle"},
		{"type":"image","fileId":"def456"},
		{"type":"image","fileId":"abc123"}
	]}`

	result := extractImageFileIDs(json.RawMessage(scene))
	if len(result) != 2 {
		t.Fatalf("expected 2 unique IDs, got %d: %v", len(result), result)
	}

	seen := make(map[string]bool)
	for _, id := range result {
		seen[id] = true
	}
	if !seen["abc123"] || !seen["def456"] {
		t.Errorf("expected abc123 and def456, got %v", result)
	}
}

func TestExtractImageFileIDs_NoFileID(t *testing.T) {
	scene := `{"elements":[{"type":"image"},{"type":"image","fileId":""}]}`
	result := extractImageFileIDs(json.RawMessage(scene))
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestExtractImageFileIDs_InvalidJSON(t *testing.T) {
	result := extractImageFileIDs(json.RawMessage(`{invalid`))
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

func TestAllowedMIMETypes(t *testing.T) {
	allowed := []string{"image/png", "image/jpeg", "image/gif", "image/webp", "image/svg+xml"}
	for _, mime := range allowed {
		if !allowedMIMETypes[mime] {
			t.Errorf("expected %s to be allowed", mime)
		}
	}

	disallowed := []string{"application/pdf", "text/html", "image/bmp", "video/mp4"}
	for _, mime := range disallowed {
		if allowedMIMETypes[mime] {
			t.Errorf("expected %s to be disallowed", mime)
		}
	}
}

func TestParseDataURL_SVG(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`
	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	dataURL := "data:image/svg+xml;base64," + encoded

	mime, data, err := parseDataURL(dataURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/svg+xml" {
		t.Errorf("expected 'image/svg+xml', got '%s'", mime)
	}
	if string(data) != svg {
		t.Errorf("SVG data mismatch")
	}
}

func TestExtractImageFileIDs_OnlyImages(t *testing.T) {
	scene := `{"elements":[
		{"type":"image","fileId":"a"},
		{"type":"image","fileId":"b"},
		{"type":"image","fileId":"c"}
	]}`

	result := extractImageFileIDs(json.RawMessage(scene))
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
}
