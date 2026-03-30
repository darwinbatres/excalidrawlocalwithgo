package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func newTestHub() *Hub {
	return NewHub(HubConfig{
		MaxConnsPerBoard:  50,
		HeartbeatInterval: 30 * time.Second,
		CursorInterval:    50 * time.Millisecond,
	}, zerolog.Nop())
}

func TestHub_GetOrCreateRoom_CreatesNew(t *testing.T) {
	h := newTestHub()

	room := h.GetOrCreateRoom("board-1")
	if room == nil {
		t.Fatal("expected non-nil room")
	}
	if room.BoardID() != "board-1" {
		t.Errorf("expected board-1, got %s", room.BoardID())
	}
}

func TestHub_GetOrCreateRoom_ReturnsSame(t *testing.T) {
	h := newTestHub()

	r1 := h.GetOrCreateRoom("board-1")
	r2 := h.GetOrCreateRoom("board-1")
	if r1 != r2 {
		t.Error("expected same room instance for same boardID")
	}
}

func TestHub_GetOrCreateRoom_DifferentBoards(t *testing.T) {
	h := newTestHub()

	r1 := h.GetOrCreateRoom("board-1")
	r2 := h.GetOrCreateRoom("board-2")
	if r1 == r2 {
		t.Error("expected different rooms for different boardIDs")
	}
}

func TestHub_GetRoom_Exists(t *testing.T) {
	h := newTestHub()
	h.GetOrCreateRoom("board-1")

	room := h.GetRoom("board-1")
	if room == nil {
		t.Fatal("expected room to exist")
	}
}

func TestHub_GetRoom_NotExists(t *testing.T) {
	h := newTestHub()

	room := h.GetRoom("nonexistent")
	if room != nil {
		t.Error("expected nil for nonexistent room")
	}
}

func TestHub_RoomCount(t *testing.T) {
	h := newTestHub()
	if h.RoomCount() != 0 {
		t.Errorf("expected 0, got %d", h.RoomCount())
	}

	h.GetOrCreateRoom("board-1")
	h.GetOrCreateRoom("board-2")
	if h.RoomCount() != 2 {
		t.Errorf("expected 2, got %d", h.RoomCount())
	}
}

func TestHub_Stats_Empty(t *testing.T) {
	h := newTestHub()
	stats := h.Stats()
	if stats.ActiveRooms != 0 {
		t.Errorf("expected 0 rooms, got %d", stats.ActiveRooms)
	}
	if stats.TotalClients != 0 {
		t.Errorf("expected 0 clients, got %d", stats.TotalClients)
	}
}

func TestHub_Stats_WithRooms(t *testing.T) {
	h := newTestHub()
	h.GetOrCreateRoom("board-1")
	h.GetOrCreateRoom("board-2")

	stats := h.Stats()
	if stats.ActiveRooms != 2 {
		t.Errorf("expected 2 rooms, got %d", stats.ActiveRooms)
	}
	if stats.TotalClients != 0 {
		t.Errorf("expected 0 clients, got %d", stats.TotalClients)
	}
}

func TestHub_RemoveRoom_ViaCallback(t *testing.T) {
	h := newTestHub()
	room := h.GetOrCreateRoom("board-1")

	// Simulate empty room callback
	h.removeRoom("board-1")

	if h.RoomCount() != 0 {
		t.Errorf("expected 0 rooms after removal, got %d", h.RoomCount())
	}
	_ = room // room was created and removed
}

func TestHub_RemoveRoom_SkipsNonEmpty(t *testing.T) {
	h := newTestHub()
	room := h.GetOrCreateRoom("board-1")

	// Add a client so room isn't empty
	room.mu.Lock()
	room.clients[&Client{info: ViewerInfo{UserID: "u1"}}] = struct{}{}
	room.mu.Unlock()

	h.removeRoom("board-1")

	if h.RoomCount() != 1 {
		t.Errorf("expected room to remain (not empty), got %d rooms", h.RoomCount())
	}
}

func TestHub_RunAndShutdown(t *testing.T) {
	h := newTestHub()
	ctx := context.Background()

	h.Run(ctx)
	// Give goroutines a moment to start
	time.Sleep(10 * time.Millisecond)

	h.Shutdown()
	// Should not panic or hang
}

func TestHub_TotalClients_Empty(t *testing.T) {
	h := newTestHub()
	if h.TotalClients() != 0 {
		t.Errorf("expected 0 clients, got %d", h.TotalClients())
	}
}

func TestHub_TotalClients_WithClients(t *testing.T) {
	h := newTestHub()
	r1 := h.GetOrCreateRoom("board-1")
	r2 := h.GetOrCreateRoom("board-2")

	r1.mu.Lock()
	r1.clients[&Client{info: ViewerInfo{UserID: "u1"}}] = struct{}{}
	r1.clients[&Client{info: ViewerInfo{UserID: "u2"}}] = struct{}{}
	r1.mu.Unlock()

	r2.mu.Lock()
	r2.clients[&Client{info: ViewerInfo{UserID: "u3"}}] = struct{}{}
	r2.mu.Unlock()

	if h.TotalClients() != 3 {
		t.Errorf("expected 3 total clients, got %d", h.TotalClients())
	}
}
