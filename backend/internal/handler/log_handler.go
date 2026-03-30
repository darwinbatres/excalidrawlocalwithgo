package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/darwinbatres/drawgo/backend/internal/middleware"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/logbuffer"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/response"
)

// LogHandler serves backend log entries from the in-memory ring buffer.
type LogHandler struct {
	buf *logbuffer.RingBuffer
}

// NewLogHandler creates a LogHandler.
func NewLogHandler(buf *logbuffer.RingBuffer) *LogHandler {
	return &LogHandler{buf: buf}
}

// Query handles GET /api/v1/logs.
// Query params: level, search, start, end, limit, offset.
func (h *LogHandler) Query(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	q := r.URL.Query()

	limit := 100
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := 0
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		offset = v
	}

	params := logbuffer.QueryParams{
		Level:  q.Get("level"),
		Search: q.Get("search"),
		Limit:  limit,
		Offset: offset,
	}

	if s := q.Get("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			params.Start = &t
		}
	}
	if s := q.Get("end"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			params.End = &t
		}
	}

	result := h.buf.Query(params)

	total := int64(result.Total)
	response.JSONWithMeta(w, http.StatusOK, result.Entries, response.Meta{
		Total:  &total,
		Limit:  &result.Limit,
		Offset: &result.Offset,
	})
}

// Summary handles GET /api/v1/logs/summary.
func (h *LogHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		response.Err(w, r, apierror.ErrUnauthorized)
		return
	}

	summary := h.buf.Summary()
	response.JSON(w, http.StatusOK, summary)
}
