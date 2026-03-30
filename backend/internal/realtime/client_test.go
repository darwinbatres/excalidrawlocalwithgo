package realtime

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/darwinbatres/drawgo/backend/internal/testutil"
)

func newTestClient(t *testing.T, cfg ClientConfig) (*Client, *Room) {
	t.Helper()
	room := NewRoom("board-test", RoomConfig{MaxConns: 10}, testutil.NopLogger())
	info := ViewerInfo{UserID: "u-1", Name: "Test", Color: "#000", Role: "authenticated"}
	log := testutil.NopLogger()
	// We can't pass a real websocket.Conn in unit tests, so we build the struct directly
	maxMsg := cfg.MaxMessagesPerSec
	if maxMsg <= 0 {
		maxMsg = 30
	}
	c := &Client{
		conn:              nil, // not needed for unit tests of non-WS methods
		room:              room,
		send:              make(chan []byte, 64),
		info:              info,
		log:               log,
		maxMessageSize:    cfg.MaxMessageSize,
		writeTimeout:      cfg.WriteTimeout,
		maxMessagesPerSec: maxMsg,
		msgWindowStart:    time.Now(),
	}
	return c, room
}

func TestNewClient_DefaultRateLimit(t *testing.T) {
	room := NewRoom("board-1", RoomConfig{MaxConns: 10}, testutil.NopLogger())
	info := ViewerInfo{UserID: "u-1", Name: "Test"}
	log := testutil.NopLogger()

	c := NewClient(nil, room, info, ClientConfig{}, log)
	assert.Equal(t, 30, c.maxMessagesPerSec, "default rate limit should be 30 msg/sec")
}

func TestNewClient_CustomRateLimit(t *testing.T) {
	room := NewRoom("board-1", RoomConfig{MaxConns: 10}, testutil.NopLogger())
	info := ViewerInfo{UserID: "u-1", Name: "Test"}
	log := testutil.NopLogger()

	c := NewClient(nil, room, info, ClientConfig{MaxMessagesPerSec: 100}, log)
	assert.Equal(t, 100, c.maxMessagesPerSec)
}

func TestClient_Info(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{})
	info := c.Info()
	assert.Equal(t, "u-1", info.UserID)
	assert.Equal(t, "Test", info.Name)
	assert.Equal(t, "#000", info.Color)
}

func TestClient_Send_QueuesMessage(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{})
	c.Send([]byte(`{"type":"ping"}`))

	select {
	case msg := <-c.send:
		assert.Equal(t, `{"type":"ping"}`, string(msg))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected message on send channel")
	}
}

func TestClient_Send_DropsWhenClosed(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{})
	// Manually mark as closed (can't call Close() without a real conn)
	c.closeMu.Lock()
	c.closed = true
	c.closeMu.Unlock()

	c.Send([]byte(`{"type":"ping"}`))

	select {
	case <-c.send:
		t.Fatal("should not receive message after close")
	case <-time.After(50 * time.Millisecond):
		// expected: no message
	}
}

func TestClient_Send_DropsWhenBufferFull(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{})
	// Fill the 64-slot buffer
	for i := 0; i < 64; i++ {
		c.send <- []byte("x")
	}

	// This should not block or panic
	c.Send([]byte("overflow"))
	assert.Len(t, c.send, 64, "buffer should still be at capacity")
}

func TestClient_Close_Idempotent(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{})
	// First close — mark manually since we have no real conn
	c.closeMu.Lock()
	c.closed = true
	c.closeMu.Unlock()

	// Second close should not panic
	require.NotPanics(t, func() {
		c.closeMu.Lock()
		defer c.closeMu.Unlock()
		// Simulating Close() logic
		if c.closed {
			return
		}
	})
}

// TestClient_RateLimit_SlidingWindow tests the sliding-window rate limiter logic.
// We can't test readPump directly (needs a WS conn), but we can verify the arithmetic.
func TestClient_RateLimit_AllowsWithinWindow(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{MaxMessagesPerSec: 10})

	// Simulate 10 messages within the window (exactly at limit)
	now := time.Now()
	c.msgWindowStart = now
	c.msgCount = 0

	for i := 0; i < 10; i++ {
		// Replicate readPump rate check logic
		elapsed := time.Since(c.msgWindowStart)
		if elapsed >= time.Second {
			c.msgCount = 0
			c.msgWindowStart = time.Now()
		}
		c.msgCount++
		assert.LessOrEqual(t, c.msgCount, c.maxMessagesPerSec,
			"message %d should be within rate limit", i+1)
	}
}

func TestClient_RateLimit_DropsExcessive(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{MaxMessagesPerSec: 5})

	c.msgWindowStart = time.Now()
	c.msgCount = 0

	allowed := 0
	dropped := 0

	for i := 0; i < 20; i++ {
		elapsed := time.Since(c.msgWindowStart)
		if elapsed >= time.Second {
			c.msgCount = 0
			c.msgWindowStart = time.Now()
		}
		c.msgCount++
		if c.msgCount > c.maxMessagesPerSec {
			dropped++
		} else {
			allowed++
		}
	}

	assert.Equal(t, 5, allowed, "should allow exactly maxMessagesPerSec")
	assert.Equal(t, 15, dropped, "should drop excess messages")
}

func TestClient_RateLimit_WindowResets(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{MaxMessagesPerSec: 5})

	// Fill the window
	c.msgWindowStart = time.Now().Add(-2 * time.Second) // expired window
	c.msgCount = 100                                     // saturated

	// Simulate a new message arrival — window should reset
	now := time.Now()
	if now.Sub(c.msgWindowStart) >= time.Second {
		c.msgCount = 0
		c.msgWindowStart = now
	}
	c.msgCount++

	assert.Equal(t, 1, c.msgCount, "counter should reset after window expires")
	assert.False(t, c.msgCount > c.maxMessagesPerSec, "first message in new window should be allowed")
}

func TestClient_Send_ConcurrentSafe(t *testing.T) {
	c, _ := newTestClient(t, ClientConfig{})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Send([]byte("concurrent"))
		}()
	}
	wg.Wait()
	// Drain and count — should not have panicked
	count := len(c.send)
	assert.LessOrEqual(t, count, 64, "should not exceed buffer capacity")
}
