package logbuffer

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"
)

// Entry represents a single parsed log entry stored in the ring buffer.
type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Raw       string                 `json:"raw"`
}

// QueryParams controls filtering and pagination for log queries.
type QueryParams struct {
	Level  string     // Filter by minimum level (debug, info, warn, error, fatal)
	Search string     // Full-text search across message and fields
	Start  *time.Time // Start of time range (inclusive)
	End    *time.Time // End of time range (inclusive)
	Limit  int        // Page size (default 100, max 1000)
	Offset int        // Page offset
}

// QueryResult is the paginated result of a log query.
type QueryResult struct {
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
}

// LevelSummary contains count of entries per log level.
type LevelSummary struct {
	Debug int `json:"debug"`
	Info  int `json:"info"`
	Warn  int `json:"warn"`
	Error int `json:"error"`
	Fatal int `json:"fatal"`
	Total int `json:"total"`
}

// RingBuffer is a thread-safe circular buffer for structured log entries.
// It implements io.Writer so it can be used as a zerolog output target.
type RingBuffer struct {
	mu      sync.RWMutex
	entries []Entry
	size    int
	pos     int // next write position
	count   int // total entries written (can exceed size)
	output  io.Writer
}

// New creates a RingBuffer that keeps the last `size` log entries
// and tees all writes to the given output writer (e.g., os.Stdout).
func New(size int, output io.Writer) *RingBuffer {
	if size <= 0 {
		size = 10000
	}
	return &RingBuffer{
		entries: make([]Entry, size),
		size:    size,
		output:  output,
	}
}

// Write implements io.Writer. It parses structured JSON log lines produced by
// zerolog and stores them in the ring buffer, while also writing to the
// underlying output.
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	// Always write to the underlying output first
	n, err = rb.output.Write(p)
	if err != nil {
		return n, err
	}

	// Parse the JSON log line
	entry := rb.parseLine(p)
	rb.mu.Lock()
	rb.entries[rb.pos] = entry
	rb.pos = (rb.pos + 1) % rb.size
	rb.count++
	rb.mu.Unlock()

	return n, nil
}

// parseLine extracts a structured Entry from a raw JSON log line.
func (rb *RingBuffer) parseLine(p []byte) Entry {
	raw := strings.TrimSpace(string(p))
	entry := Entry{
		Timestamp: time.Now(),
		Level:     "info",
		Raw:       raw,
	}

	var m map[string]interface{}
	if err := json.Unmarshal(p, &m); err != nil {
		// Not JSON — store as raw message
		entry.Message = raw
		return entry
	}

	// Extract standard zerolog fields
	if ts, ok := m["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			entry.Timestamp = t
		} else if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			entry.Timestamp = t
		}
		delete(m, "time")
	}
	if lvl, ok := m["level"].(string); ok {
		entry.Level = lvl
		delete(m, "level")
	}
	if msg, ok := m["message"].(string); ok {
		entry.Message = msg
		delete(m, "message")
	} else if msg, ok := m["msg"].(string); ok {
		entry.Message = msg
		delete(m, "msg")
	}
	if caller, ok := m["caller"].(string); ok {
		entry.Caller = caller
		delete(m, "caller")
	}

	if len(m) > 0 {
		entry.Fields = m
	}

	return entry
}

// Query returns log entries matching the given parameters.
func (rb *RingBuffer) Query(params QueryParams) QueryResult {
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	minLevel := levelPriority(params.Level)
	searchLower := strings.ToLower(params.Search)

	rb.mu.RLock()
	filled := rb.filledCount()

	// Collect matching entries in reverse chronological order (newest first)
	matched := make([]Entry, 0, min(filled, params.Limit+params.Offset))
	for i := 0; i < filled; i++ {
		idx := rb.reverseIndex(i)
		e := rb.entries[idx]

		if !rb.matchEntry(e, minLevel, searchLower, params.Start, params.End) {
			continue
		}
		matched = append(matched, e)
	}
	rb.mu.RUnlock()

	total := len(matched)

	// Apply pagination
	start := params.Offset
	if start > total {
		start = total
	}
	end := start + params.Limit
	if end > total {
		end = total
	}

	return QueryResult{
		Entries: matched[start:end],
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
	}
}

// Summary returns counts per log level for the current buffer contents.
func (rb *RingBuffer) Summary() LevelSummary {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	filled := rb.filledCount()
	var s LevelSummary
	for i := 0; i < filled; i++ {
		idx := rb.reverseIndex(i)
		switch rb.entries[idx].Level {
		case "debug", "trace":
			s.Debug++
		case "info":
			s.Info++
		case "warn", "warning":
			s.Warn++
		case "error":
			s.Error++
		case "fatal", "panic":
			s.Fatal++
		}
	}
	s.Total = filled
	return s
}

// filledCount returns how many slots are actually filled. Must be called with lock held.
func (rb *RingBuffer) filledCount() int {
	if rb.count >= rb.size {
		return rb.size
	}
	return rb.count
}

// reverseIndex returns the buffer index for the i-th most recent entry.
// Must be called with lock held.
func (rb *RingBuffer) reverseIndex(i int) int {
	return (rb.pos - 1 - i + rb.size) % rb.size
}

// matchEntry checks if an entry passes all filters.
func (rb *RingBuffer) matchEntry(e Entry, minLevel int, searchLower string, start, end *time.Time) bool {
	if minLevel > 0 && levelPriority(e.Level) < minLevel {
		return false
	}
	if start != nil && e.Timestamp.Before(*start) {
		return false
	}
	if end != nil && e.Timestamp.After(*end) {
		return false
	}
	if searchLower != "" {
		if !strings.Contains(strings.ToLower(e.Message), searchLower) &&
			!strings.Contains(strings.ToLower(e.Raw), searchLower) {
			return false
		}
	}
	return true
}

// levelPriority maps log level strings to numeric priorities for filtering.
func levelPriority(level string) int {
	switch strings.ToLower(level) {
	case "trace", "debug":
		return 1
	case "info":
		return 2
	case "warn", "warning":
		return 3
	case "error":
		return 4
	case "fatal", "panic":
		return 5
	default:
		return 0 // no filter
	}
}
