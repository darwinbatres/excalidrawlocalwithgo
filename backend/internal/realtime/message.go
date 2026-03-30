package realtime

import (
	"encoding/json"
	"time"
)

// MessageType identifies the kind of WebSocket message.
type MessageType string

const (
	// Client → Server messages
	MsgTypeCursorMove  MessageType = "cursor_move"
	MsgTypeSceneUpdate MessageType = "scene_update"
	MsgTypePing        MessageType = "ping"

	// Server → Client messages
	MsgTypePong         MessageType = "pong"
	MsgTypeJoined       MessageType = "joined"
	MsgTypeLeft         MessageType = "left"
	MsgTypePresence     MessageType = "presence"
	MsgTypeWelcome      MessageType = "welcome"
	MsgTypeError        MessageType = "error"
	MsgTypeBroadcast    MessageType = "broadcast"
	MsgTypeSceneSynced  MessageType = "scene_synced"
	MsgTypeCursorUpdate MessageType = "cursor_update"
)

// Message is the wire format for all WebSocket communication.
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	SenderID string         `json:"senderId,omitempty"`
}

// CursorPayload represents a user's cursor position.
type CursorPayload struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	UserID string  `json:"userId"`
	Name   string  `json:"name,omitempty"`
	Color  string  `json:"color,omitempty"`
}

// PresencePayload is sent when the viewer list changes.
type PresencePayload struct {
	Viewers []ViewerInfo `json:"viewers"`
	Count   int          `json:"count"`
}

// ViewerInfo describes a connected participant.
type ViewerInfo struct {
	UserID    string `json:"userId"`
	Name      string `json:"name,omitempty"`
	Color     string `json:"color,omitempty"`
	Role      string `json:"role"`
	JoinedAt  int64  `json:"joinedAt"`
	IsAnon    bool   `json:"isAnon"`
}

// WelcomePayload is sent to the connecting client with their own identity.
type WelcomePayload struct {
	Viewer ViewerInfo `json:"viewer"`
}

// JoinedPayload is sent to all clients when a user joins.
type JoinedPayload struct {
	Viewer ViewerInfo `json:"viewer"`
}

// LeftPayload is sent to all clients when a user leaves.
type LeftPayload struct {
	UserID string `json:"userId"`
}

// ErrorPayload describes a server-side error.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// marshalPayload is a helper for encoding a typed payload into a Message.
func NewMessage(msgType MessageType, senderID string, payload any) (*Message, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return &Message{
		Type:     msgType,
		Payload:  raw,
		SenderID: senderID,
	}, nil
}

// MustMessage is like NewMessage but panics on error (for constant payloads).
func MustMessage(msgType MessageType, senderID string, payload any) *Message {
	m, err := NewMessage(msgType, senderID, payload)
	if err != nil {
		panic(err)
	}
	return m
}

// cursorColors is a palette for assigning colors to anonymous viewers.
var cursorColors = []string{
	"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
	"#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
	"#BB8FCE", "#85C1E9", "#F0B27A", "#82E0AA",
}

// AssignColor picks a cursor color based on a deterministic index.
func AssignColor(index int) string {
	return cursorColors[index%len(cursorColors)]
}

// nowUnixMs returns the current time as Unix milliseconds.
func nowUnixMs() int64 {
	return time.Now().UnixMilli()
}
