package realtime

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func newTestRoom(maxConns int) *Room {
	return NewRoom("board-1", RoomConfig{
		MaxConns:       maxConns,
		CursorInterval: 50 * time.Millisecond,
	}, zerolog.Nop())
}

func TestRoom_BoardID(t *testing.T) {
	r := newTestRoom(10)
	if r.BoardID() != "board-1" {
		t.Errorf("expected board-1, got %s", r.BoardID())
	}
}

func TestRoom_ClientCount_Empty(t *testing.T) {
	r := newTestRoom(10)
	if r.ClientCount() != 0 {
		t.Errorf("expected 0, got %d", r.ClientCount())
	}
}

func TestRoom_IsFull_Unlimited(t *testing.T) {
	r := newTestRoom(0) // 0 = unlimited
	if r.IsFull() {
		t.Error("expected not full when maxConns=0")
	}
}

func TestRoom_IsFull_AtCapacity(t *testing.T) {
	r := newTestRoom(2)
	// Manually add clients to test capacity
	r.mu.Lock()
	r.clients[&Client{info: ViewerInfo{UserID: "u1"}}] = struct{}{}
	r.clients[&Client{info: ViewerInfo{UserID: "u2"}}] = struct{}{}
	r.mu.Unlock()

	if !r.IsFull() {
		t.Error("expected full when at capacity")
	}
}

func TestRoom_IsFull_BelowCapacity(t *testing.T) {
	r := newTestRoom(5)
	r.mu.Lock()
	r.clients[&Client{info: ViewerInfo{UserID: "u1"}}] = struct{}{}
	r.mu.Unlock()

	if r.IsFull() {
		t.Error("expected not full when below capacity")
	}
}

func TestRoom_NextColor_Sequential(t *testing.T) {
	r := newTestRoom(10)

	c0 := r.NextColor()
	c1 := r.NextColor()
	c2 := r.NextColor()

	if c0 != cursorColors[0] {
		t.Errorf("expected %s, got %s", cursorColors[0], c0)
	}
	if c1 != cursorColors[1] {
		t.Errorf("expected %s, got %s", cursorColors[1], c1)
	}
	if c2 != cursorColors[2] {
		t.Errorf("expected %s, got %s", cursorColors[2], c2)
	}
}

func TestRoom_NextColor_Wraps(t *testing.T) {
	r := newTestRoom(10)

	// Exhaust the palette
	for i := 0; i < len(cursorColors); i++ {
		r.NextColor()
	}

	// Should wrap to first color
	color := r.NextColor()
	if color != cursorColors[0] {
		t.Errorf("expected wrap to %s, got %s", cursorColors[0], color)
	}
}

func TestRoom_FlushCursors_EmptyBuffer(t *testing.T) {
	r := newTestRoom(10)
	// Should not panic or send when buffer is empty
	r.FlushCursors()
}

func TestRoom_FlushCursors_ClearsBuffer(t *testing.T) {
	r := newTestRoom(10)

	// Add cursor data to buffer
	r.cursorMu.Lock()
	r.cursorBuf["u1"] = &CursorPayload{X: 10, Y: 20, UserID: "u1"}
	r.cursorBuf["u2"] = &CursorPayload{X: 30, Y: 40, UserID: "u2"}
	r.cursorMu.Unlock()

	r.FlushCursors()

	r.cursorMu.Lock()
	remaining := len(r.cursorBuf)
	r.cursorMu.Unlock()

	if remaining != 0 {
		t.Errorf("expected empty buffer after flush, got %d entries", remaining)
	}
}

func TestRoom_OnEmpty_Callback(t *testing.T) {
	called := false
	calledWith := ""

	r := NewRoom("board-empty", RoomConfig{
		MaxConns:       10,
		CursorInterval: 50 * time.Millisecond,
		OnEmpty: func(boardID string) {
			called = true
			calledWith = boardID
		},
	}, zerolog.Nop())

	// Add then remove a client
	c := &Client{
		info: ViewerInfo{UserID: "u1"},
		send: make(chan []byte, 1),
	}
	r.mu.Lock()
	r.clients[c] = struct{}{}
	r.mu.Unlock()

	r.Leave(c)

	if !called {
		t.Error("expected onEmpty callback to be called")
	}
	if calledWith != "board-empty" {
		t.Errorf("expected board-empty, got %s", calledWith)
	}
}

func TestRoom_Leave_Idempotent(t *testing.T) {
	callCount := 0
	r := NewRoom("board-idem", RoomConfig{
		MaxConns:       10,
		CursorInterval: 50 * time.Millisecond,
		OnEmpty: func(boardID string) {
			callCount++
		},
	}, zerolog.Nop())

	c := &Client{
		info: ViewerInfo{UserID: "u1"},
		send: make(chan []byte, 1),
	}
	r.mu.Lock()
	r.clients[c] = struct{}{}
	r.mu.Unlock()

	r.Leave(c)
	r.Leave(c) // second call should be no-op

	if callCount != 1 {
		t.Errorf("expected onEmpty called once, got %d", callCount)
	}
}
