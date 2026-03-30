package handler

import (
	"encoding/json"
	"strconv"
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

// parsePagination extracts limit/offset from query params with defaults and bounds.
func parsePagination(limitStr, offsetStr string) (int, int) {
	limit := defaultLimit
	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
		limit = v
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := 0
	if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
		offset = v
	}

	return limit, offset
}

// marshalToRaw converts an interface{} to json.RawMessage.
func marshalToRaw(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
