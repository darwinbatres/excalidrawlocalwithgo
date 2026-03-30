package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"
)

// Client represents a single WebSocket connection to a board room.
// Each client runs two goroutines: readPump and writePump.
type Client struct {
	conn    *websocket.Conn
	room    *Room
	send    chan []byte
	info    ViewerInfo
	log     zerolog.Logger
	cancel  context.CancelFunc
	closed  bool
	closeMu sync.Mutex

	// Configuration
	maxMessageSize    int64
	writeTimeout      time.Duration
	maxMessagesPerSec int

	// Rate limiting state (simple sliding-window counter)
	msgCount       int
	sceneCount     int
	msgWindowStart time.Time
}

// ClientConfig holds per-connection configuration.
type ClientConfig struct {
	MaxMessageSize    int64
	WriteTimeout      time.Duration
	MaxMessagesPerSec int // 0 = unlimited
}

// NewClient creates a Client for a WebSocket connection.
func NewClient(conn *websocket.Conn, room *Room, info ViewerInfo, cfg ClientConfig, log zerolog.Logger) *Client {
	maxMsg := cfg.MaxMessagesPerSec
	if maxMsg <= 0 {
		maxMsg = 30 // default: 30 messages/sec
	}
	return &Client{
		conn:              conn,
		room:              room,
		send:              make(chan []byte, 64),
		info:              info,
		log:               log.With().Str("userId", info.UserID).Str("room", room.BoardID()).Logger(),
		maxMessageSize:    cfg.MaxMessageSize,
		writeTimeout:      cfg.WriteTimeout,
		maxMessagesPerSec: maxMsg,
		msgWindowStart:    time.Now(),
	}
}

// Info returns the viewer info for this client.
func (c *Client) Info() ViewerInfo {
	return c.info
}

// Send queues a message for the client's write pump. Non-blocking — drops if buffer full or client closed.
func (c *Client) Send(data []byte) {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return
	}
	c.closeMu.Unlock()

	select {
	case c.send <- data:
	default:
		c.log.Warn().Msg("client send buffer full, dropping message")
	}
}

// Run starts the read and write pumps. Blocks until the connection closes.
func (c *Client) Run(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	defer c.Close()

	go c.writePump(ctx)
	c.readPump(ctx)
}

// Close cleanly shuts down the client.
func (c *Client) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return
	}
	c.closed = true

	if c.cancel != nil {
		c.cancel()
	}
	c.conn.Close(websocket.StatusNormalClosure, "")
	c.room.Leave(c)
}

// readPump reads messages from the WebSocket and dispatches them to the room.
func (c *Client) readPump(ctx context.Context) {
	c.conn.SetReadLimit(c.maxMessageSize)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				c.log.Debug().Msg("client disconnected")
			} else {
				c.log.Debug().Err(err).Msg("read error")
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.log.Debug().Err(err).Msg("invalid message format")
			continue
		}

		// Per-client rate limiting (sliding window) — applies only to
		// high-frequency messages like cursor_move. Critical sync messages
		// (scene_update, ping) are always forwarded so real-time collaboration
		// keeps working even when cursor traffic is heavy. Scene updates get
		// a separate, generous limit to prevent abuse.
		now := time.Now()
		if now.Sub(c.msgWindowStart) >= time.Second {
			c.msgCount = 0
			c.sceneCount = 0
			c.msgWindowStart = now
		}
		switch msg.Type {
		case MsgTypeCursorMove:
			c.msgCount++
			if c.msgCount > c.maxMessagesPerSec {
				continue
			}
		case MsgTypeSceneUpdate:
			c.sceneCount++
			if c.sceneCount > 5 { // max 5 scene updates per second per client
				continue
			}
		}

		msg.SenderID = c.info.UserID
		c.room.HandleMessage(c, &msg)
	}
}

// writePump writes queued messages to the WebSocket connection.
func (c *Client) writePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, writeCancel := context.WithTimeout(ctx, c.writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, data)
			writeCancel()
			if err != nil {
				c.log.Debug().Err(err).Msg("write error")
				return
			}
		}
	}
}
