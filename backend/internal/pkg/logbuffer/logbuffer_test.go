package logbuffer

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestRingBuffer_Write(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	line := `{"time":"2025-01-15T10:00:00Z","level":"info","message":"hello world"}` + "\n"
	n, err := buf.Write([]byte(line))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(line) {
		t.Fatalf("wrote %d, want %d", n, len(line))
	}

	// Verify it was also written to the underlying output
	if out.String() != line {
		t.Fatalf("output mismatch: %q", out.String())
	}

	// Query and verify
	result := buf.Query(QueryParams{Limit: 10})
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if result.Entries[0].Message != "hello world" {
		t.Fatalf("message = %q, want %q", result.Entries[0].Message, "hello world")
	}
	if result.Entries[0].Level != "info" {
		t.Fatalf("level = %q, want %q", result.Entries[0].Level, "info")
	}
}

func TestRingBuffer_Wrapping(t *testing.T) {
	var out bytes.Buffer
	buf := New(5, &out) // tiny buffer

	for i := 0; i < 10; i++ {
		line, _ := json.Marshal(map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "info",
			"message": i,
		})
		line = append(line, '\n')
		if _, err := buf.Write(line); err != nil {
			t.Fatal(err)
		}
	}

	result := buf.Query(QueryParams{Limit: 100})
	if result.Total != 5 {
		t.Fatalf("total = %d, want 5 (ring wrapped)", result.Total)
	}
}

func TestRingBuffer_LevelFilter(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	levels := []string{"debug", "info", "warn", "error"}
	for _, lvl := range levels {
		line, _ := json.Marshal(map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   lvl,
			"message": lvl + " msg",
		})
		line = append(line, '\n')
		buf.Write(line)
	}

	// Filter: warn and above
	result := buf.Query(QueryParams{Level: "warn", Limit: 100})
	if result.Total != 2 {
		t.Fatalf("total = %d, want 2 (warn + error)", result.Total)
	}

	// Filter: error only
	result = buf.Query(QueryParams{Level: "error", Limit: 100})
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1 (error only)", result.Total)
	}
}

func TestRingBuffer_Search(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	messages := []string{"database connected", "user logged in", "database query failed"}
	for _, msg := range messages {
		line, _ := json.Marshal(map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "info",
			"message": msg,
		})
		line = append(line, '\n')
		buf.Write(line)
	}

	result := buf.Query(QueryParams{Search: "database", Limit: 100})
	if result.Total != 2 {
		t.Fatalf("total = %d, want 2", result.Total)
	}

	result = buf.Query(QueryParams{Search: "LOGGED", Limit: 100}) // case-insensitive
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
}

func TestRingBuffer_TimeRange(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	base := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		line, _ := json.Marshal(map[string]interface{}{
			"time":    ts.Format(time.RFC3339),
			"level":   "info",
			"message": ts.Format("15:04"),
		})
		line = append(line, '\n')
		buf.Write(line)
	}

	start := base.Add(1 * time.Hour)
	end := base.Add(3 * time.Hour)
	result := buf.Query(QueryParams{Start: &start, End: &end, Limit: 100})
	if result.Total != 3 {
		t.Fatalf("total = %d, want 3 (hours 1-3)", result.Total)
	}
}

func TestRingBuffer_Pagination(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	for i := 0; i < 20; i++ {
		line, _ := json.Marshal(map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "info",
			"message": i,
		})
		line = append(line, '\n')
		buf.Write(line)
	}

	// Page 1
	result := buf.Query(QueryParams{Limit: 5, Offset: 0})
	if len(result.Entries) != 5 {
		t.Fatalf("page1 len = %d, want 5", len(result.Entries))
	}
	if result.Total != 20 {
		t.Fatalf("total = %d, want 20", result.Total)
	}

	// Page 2
	result = buf.Query(QueryParams{Limit: 5, Offset: 5})
	if len(result.Entries) != 5 {
		t.Fatalf("page2 len = %d, want 5", len(result.Entries))
	}

	// Beyond end
	result = buf.Query(QueryParams{Limit: 5, Offset: 20})
	if len(result.Entries) != 0 {
		t.Fatalf("beyond end len = %d, want 0", len(result.Entries))
	}
}

func TestRingBuffer_Summary(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	data := []struct {
		level string
		count int
	}{
		{"debug", 3},
		{"info", 5},
		{"warn", 2},
		{"error", 1},
	}

	for _, d := range data {
		for i := 0; i < d.count; i++ {
			line, _ := json.Marshal(map[string]interface{}{
				"time":    time.Now().Format(time.RFC3339),
				"level":   d.level,
				"message": "test",
			})
			line = append(line, '\n')
			buf.Write(line)
		}
	}

	s := buf.Summary()
	if s.Debug != 3 {
		t.Fatalf("debug = %d, want 3", s.Debug)
	}
	if s.Info != 5 {
		t.Fatalf("info = %d, want 5", s.Info)
	}
	if s.Warn != 2 {
		t.Fatalf("warn = %d, want 2", s.Warn)
	}
	if s.Error != 1 {
		t.Fatalf("error = %d, want 1", s.Error)
	}
	if s.Total != 11 {
		t.Fatalf("total = %d, want 11", s.Total)
	}
}

func TestRingBuffer_NonJSON(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	line := "plain text log line\n"
	buf.Write([]byte(line))

	result := buf.Query(QueryParams{Limit: 10})
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if result.Entries[0].Message != "plain text log line" {
		t.Fatalf("message = %q", result.Entries[0].Message)
	}
}

func TestRingBuffer_NewestFirst(t *testing.T) {
	var out bytes.Buffer
	buf := New(100, &out)

	for i := 0; i < 3; i++ {
		ts := time.Date(2025, 1, 15, 10+i, 0, 0, 0, time.UTC)
		line, _ := json.Marshal(map[string]interface{}{
			"time":    ts.Format(time.RFC3339),
			"level":   "info",
			"message": ts.Format("15:04"),
		})
		line = append(line, '\n')
		buf.Write(line)
	}

	result := buf.Query(QueryParams{Limit: 10})
	if len(result.Entries) != 3 {
		t.Fatalf("len = %d, want 3", len(result.Entries))
	}
	// Newest first
	if result.Entries[0].Message != "12:00" {
		t.Fatalf("first entry = %q, want 12:00 (newest)", result.Entries[0].Message)
	}
	if result.Entries[2].Message != "10:00" {
		t.Fatalf("last entry = %q, want 10:00 (oldest)", result.Entries[2].Message)
	}
}
