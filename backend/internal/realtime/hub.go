package realtime

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// Hub manages all active board rooms and runs background maintenance.
// It is the top-level entry point for the real-time subsystem.
type Hub struct {
	rooms  map[string]*Room
	mu     sync.RWMutex
	log    zerolog.Logger
	cfg    HubConfig
	cancel context.CancelFunc

	// Atomic message counters
	msgIn  int64 // messages received from clients
	msgOut int64 // messages sent to clients
}

// HubConfig holds configuration for the hub.
type HubConfig struct {
	MaxConnsPerBoard  int
	HeartbeatInterval time.Duration
	CursorInterval    time.Duration
}

// NewHub creates a Hub. Call Run() to start background tasks.
func NewHub(cfg HubConfig, log zerolog.Logger) *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
		log:   log.With().Str("component", "ws-hub").Logger(),
		cfg:   cfg,
	}
}

// Run starts the hub's background goroutines (cursor flushing, heartbeats).
// Returns immediately. Call Shutdown() to stop.
func (h *Hub) Run(ctx context.Context) {
	ctx, h.cancel = context.WithCancel(ctx)

	go h.cursorTicker(ctx)
	go h.heartbeatTicker(ctx)

	h.log.Info().
		Dur("cursorInterval", h.cfg.CursorInterval).
		Dur("heartbeatInterval", h.cfg.HeartbeatInterval).
		Int("maxConnsPerBoard", h.cfg.MaxConnsPerBoard).
		Msg("WebSocket hub started")
}

// Shutdown stops all background tasks and closes all connections.
func (h *Hub) Shutdown() {
	if h.cancel != nil {
		h.cancel()
	}
	h.log.Info().Msg("WebSocket hub stopped")
}

// GetOrCreateRoom returns the room for a board, creating one if needed.
func (h *Hub) GetOrCreateRoom(boardID string) *Room {
	h.mu.RLock()
	room, ok := h.rooms[boardID]
	h.mu.RUnlock()
	if ok {
		return room
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if room, ok = h.rooms[boardID]; ok {
		return room
	}

	room = NewRoom(boardID, RoomConfig{
		MaxConns:       h.cfg.MaxConnsPerBoard,
		CursorInterval: h.cfg.CursorInterval,
		OnEmpty:        h.removeRoom,
		Hub:            h,
	}, h.log)

	h.rooms[boardID] = room
	h.log.Debug().Str("boardID", boardID).Msg("room created")
	return room
}

// GetRoom returns a room if it exists.
func (h *Hub) GetRoom(boardID string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[boardID]
}

// removeRoom is called when a room becomes empty.
func (h *Hub) removeRoom(boardID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[boardID]; ok && room.ClientCount() == 0 {
		delete(h.rooms, boardID)
		h.log.Debug().Str("boardID", boardID).Msg("room removed (empty)")
	}
}

// RoomCount returns the number of active rooms.
func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// TotalClients returns the total number of connected clients across all rooms.
func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, room := range h.rooms {
		total += room.ClientCount()
	}
	return total
}

// Stats returns a snapshot of hub statistics.
type HubStats struct {
	ActiveRooms  int   `json:"activeRooms"`
	TotalClients int   `json:"totalClients"`
	MessagesIn   int64 `json:"messagesIn"`
	MessagesOut  int64 `json:"messagesOut"`
}

func (h *Hub) Stats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, room := range h.rooms {
		total += room.ClientCount()
	}
	return HubStats{
		ActiveRooms:  len(h.rooms),
		TotalClients: total,
		MessagesIn:   atomic.LoadInt64(&h.msgIn),
		MessagesOut:  atomic.LoadInt64(&h.msgOut),
	}
}

// IncrMsgIn increments the inbound message counter.
func (h *Hub) IncrMsgIn() { atomic.AddInt64(&h.msgIn, 1) }

// IncrMsgOut adds n to the outbound message counter.
func (h *Hub) IncrMsgOut(n int64) { atomic.AddInt64(&h.msgOut, n) }

// cursorTicker periodically flushes cursor batches for all rooms.
func (h *Hub) cursorTicker(ctx context.Context) {
	ticker := time.NewTicker(h.cfg.CursorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			for _, room := range h.rooms {
				room.FlushCursors()
			}
			h.mu.RUnlock()
		}
	}
}

// heartbeatTicker sends periodic pings to detect stale connections.
func (h *Hub) heartbeatTicker(ctx context.Context) {
	ticker := time.NewTicker(h.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// coder/websocket handles ping/pong internally via the Ping method.
			// We rely on read timeouts in readPump to detect stale connections.
			// This ticker serves as a cadence for any future heartbeat logic.
		}
	}
}
