package realtime

import (
	"encoding/json"
	"testing"
)

func TestNewMessage_WithPayload(t *testing.T) {
	payload := CursorPayload{X: 10.5, Y: 20.3, UserID: "u1"}
	msg, err := NewMessage(MsgTypeCursorMove, "u1", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != MsgTypeCursorMove {
		t.Errorf("expected type %s, got %s", MsgTypeCursorMove, msg.Type)
	}
	if msg.SenderID != "u1" {
		t.Errorf("expected senderId u1, got %s", msg.SenderID)
	}
	if msg.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
	var decoded CursorPayload
	if err := json.Unmarshal(msg.Payload, &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded.X != 10.5 || decoded.Y != 20.3 {
		t.Errorf("unexpected payload values: %+v", decoded)
	}
}

func TestNewMessage_NilPayload(t *testing.T) {
	msg, err := NewMessage(MsgTypePong, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Payload != nil {
		t.Errorf("expected nil payload, got %s", string(msg.Payload))
	}
}

func TestMustMessage_Success(t *testing.T) {
	msg := MustMessage(MsgTypeJoined, "s1", JoinedPayload{
		Viewer: ViewerInfo{UserID: "u1", Name: "Alice"},
	})
	if msg.Type != MsgTypeJoined {
		t.Errorf("expected type %s, got %s", MsgTypeJoined, msg.Type)
	}
}

func TestMustMessage_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unmarshalable payload")
		}
	}()
	MustMessage(MsgTypeError, "", make(chan int))
}

func TestAssignColor_CyclesPalette(t *testing.T) {
	paletteSize := len(cursorColors)
	for i := 0; i < paletteSize; i++ {
		color := AssignColor(i)
		if color != cursorColors[i] {
			t.Errorf("index %d: expected %s, got %s", i, cursorColors[i], color)
		}
	}
	if AssignColor(paletteSize) != cursorColors[0] {
		t.Errorf("expected wrap to index 0")
	}
	if AssignColor(paletteSize+1) != cursorColors[1] {
		t.Errorf("expected wrap to index 1")
	}
}

func TestAssignColor_AllDistinct(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < len(cursorColors); i++ {
		c := AssignColor(i)
		if seen[c] {
			t.Errorf("duplicate color at index %d: %s", i, c)
		}
		seen[c] = true
	}
}

func TestNowUnixMs_ReturnsMilliseconds(t *testing.T) {
	ms := nowUnixMs()
	if ms < 1577836800000 {
		t.Errorf("timestamp too small: %d", ms)
	}
}

func TestMessage_JSONRoundTrip(t *testing.T) {
	original := MustMessage(MsgTypePresence, "sender1", PresencePayload{
		Viewers: []ViewerInfo{{UserID: "u1", Name: "Alice", Role: "VIEWER"}},
		Count:   1,
	})
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Type != MsgTypePresence {
		t.Errorf("expected type %s, got %s", MsgTypePresence, decoded.Type)
	}
	if decoded.SenderID != "sender1" {
		t.Errorf("expected senderId sender1, got %s", decoded.SenderID)
	}
}

func TestErrorPayload_JSONRoundTrip(t *testing.T) {
	msg := MustMessage(MsgTypeError, "", ErrorPayload{Code: "RATE_LIMIT", Message: "Too fast"})
	data, _ := json.Marshal(msg)
	var decoded Message
	json.Unmarshal(data, &decoded)
	var errPayload ErrorPayload
	if err := json.Unmarshal(decoded.Payload, &errPayload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if errPayload.Code != "RATE_LIMIT" {
		t.Errorf("expected code RATE_LIMIT, got %s", errPayload.Code)
	}
}
