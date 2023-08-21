package ws

import (
	"context"
	"encoding/json"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(ctx context.Context, evt Event, c *Client) error

const (
	EventSendMessage = "send_message"
	EventJoinRoom    = "join_room"
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}
