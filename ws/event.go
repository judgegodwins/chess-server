package ws

import (
	"context"
	"encoding/json"
	"fmt"
)

type Event struct {
	Type    string          `json:"type"`
	TraceID string          `json:"trace_id"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(ctx context.Context, evt Event, c *Client) error

const (
	EventSendMessage    = "send_message"
	EventJoinRoom       = "join_room"
	EventAcceptJoin     = "accept_join_request"
	EventPieceMove      = "piece_move"
	EventError          = "error"
	EventUserDisconnect = "user_disconnect"
	EventUserConnect    = "user_connect"
	EventRoomNotFound   = "room_not_found"
	EventRequestJoin    = "request_join"
	EventStartGame      = "start_game"
	EventCloseRoom      = "close_room"
	EventClosingRoom = "closing_room"
)

type PayloadError struct {
	Message string `json:"message"`
}

type PayloadSendMessage struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type PayloadRoom struct {
	RoomID string `json:"room_id"`
}

type PayloadAcceptJoinRequest struct {
	RoomID   string `json:"room_id"`
	ClientID string `json:"client_id"`
	PlayerID string `json:"player_id"`
}

type PayloadPieceMove struct {
	RoomID string          `json:"room_id"`
	Fen    string          `json:"fen"`
	Move   json.RawMessage `json:"move"`
}

type PayloadUser struct {
	UserID string `json:"user_id"`
}

func NewEvent(evtType string, payload any) (Event, error) {
	b, err := json.Marshal(payload)

	if err != nil {
		return Event{}, err
	}

	evt := NewEventStruct(evtType, b, "")

	return evt, nil
}

func NewErrorEvent(traceId, message string) (Event, error) {
	payload := PayloadError{Message: message}
	b, err := json.Marshal(payload)

	if err != nil {
		return Event{}, err
	}

	evt := NewEventStruct(fmt.Sprintf("%v_%v", EventError, traceId), b, traceId)

	return evt, nil
}

func NewEventStruct(evtType string, payload []byte, traceId string) Event {
	return Event{
		Type:    evtType,
		TraceID: traceId,
		Payload: payload,
	}
}
