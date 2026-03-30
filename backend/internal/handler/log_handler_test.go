package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/darwinbatres/drawgo/backend/internal/handler"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/logbuffer"
)

func seedLogBuffer(t *testing.T, buf *logbuffer.RingBuffer, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		levels := []string{"debug", "info", "warn", "error"}
		line, _ := json.Marshal(map[string]interface{}{
			"time":    time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			"level":   levels[i%len(levels)],
			"message": "log entry " + string(rune('A'+i%26)),
		})
		line = append(line, '\n')
		buf.Write(line)
	}
}

func TestLogHandler_Query_Unauthenticated(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})
	h := handler.NewLogHandler(buf)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	rec := httptest.NewRecorder()

	h.Query(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogHandler_Query_Basic(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})
	seedLogBuffer(t, buf, 20)

	h := handler.NewLogHandler(buf)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=5", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.Query(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []logbuffer.Entry `json:"data"`
		Meta struct {
			Total  int `json:"total"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body.Data, 5)
	assert.Equal(t, 20, body.Meta.Total)
	assert.Equal(t, 5, body.Meta.Limit)
	assert.Equal(t, 0, body.Meta.Offset)
}

func TestLogHandler_Query_WithLevel(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})
	seedLogBuffer(t, buf, 20) // 5 debug, 5 info, 5 warn, 5 error

	h := handler.NewLogHandler(buf)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?level=error&limit=100", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.Query(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []logbuffer.Entry `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 5, body.Meta.Total) // 5 error entries
}

func TestLogHandler_Query_WithSearch(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})

	// Write specific messages
	for _, msg := range []string{"database connected", "user login", "database error"} {
		line, _ := json.Marshal(map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "info",
			"message": msg,
		})
		line = append(line, '\n')
		buf.Write(line)
	}

	h := handler.NewLogHandler(buf)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?search=database&limit=100", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.Query(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []logbuffer.Entry `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 2, body.Meta.Total)
}

func TestLogHandler_Query_Pagination(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})
	seedLogBuffer(t, buf, 15)

	h := handler.NewLogHandler(buf)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=5&offset=5", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.Query(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []logbuffer.Entry `json:"data"`
		Meta struct {
			Total  int `json:"total"`
			Offset int `json:"offset"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body.Data, 5)
	assert.Equal(t, 15, body.Meta.Total)
	assert.Equal(t, 5, body.Meta.Offset)
}

func TestLogHandler_Summary(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})
	seedLogBuffer(t, buf, 8) // 2 debug, 2 info, 2 warn, 2 error

	h := handler.NewLogHandler(buf)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/summary", nil)
	req = withUserID(req, "user-1")
	rec := httptest.NewRecorder()

	h.Summary(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data logbuffer.LevelSummary `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 2, body.Data.Debug)
	assert.Equal(t, 2, body.Data.Info)
	assert.Equal(t, 2, body.Data.Warn)
	assert.Equal(t, 2, body.Data.Error)
	assert.Equal(t, 8, body.Data.Total)
}

func TestLogHandler_Summary_Unauthenticated(t *testing.T) {
	buf := logbuffer.New(100, &bytes.Buffer{})
	h := handler.NewLogHandler(buf)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/summary", nil)
	rec := httptest.NewRecorder()

	h.Summary(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
