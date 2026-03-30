package handler

import (
	"testing"
)

func TestParsePagination_Defaults(t *testing.T) {
	limit, offset := parsePagination("", "")
	if limit != defaultLimit {
		t.Errorf("expected default limit %d, got %d", defaultLimit, limit)
	}
	if offset != 0 {
		t.Errorf("expected offset 0, got %d", offset)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	limit, offset := parsePagination("25", "10")
	if limit != 25 {
		t.Errorf("expected limit 25, got %d", limit)
	}
	if offset != 10 {
		t.Errorf("expected offset 10, got %d", offset)
	}
}

func TestParsePagination_MaxLimit(t *testing.T) {
	limit, _ := parsePagination("500", "0")
	if limit != maxLimit {
		t.Errorf("expected max limit %d, got %d", maxLimit, limit)
	}
}

func TestParsePagination_InvalidValues(t *testing.T) {
	limit, offset := parsePagination("abc", "-5")
	if limit != defaultLimit {
		t.Errorf("expected default limit, got %d", limit)
	}
	if offset != 0 {
		t.Errorf("expected offset 0 for negative value, got %d", offset)
	}
}

func TestMarshalToRaw_Nil(t *testing.T) {
	raw, err := marshalToRaw(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if raw != nil {
		t.Errorf("expected nil for nil input")
	}
}

func TestMarshalToRaw_Object(t *testing.T) {
	raw, err := marshalToRaw(map[string]string{"key": "value"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(raw) != `{"key":"value"}` {
		t.Errorf("unexpected result: %s", string(raw))
	}
}
