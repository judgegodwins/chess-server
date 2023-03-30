package websocketconnection

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	ErrorEventMessage      = "error"
	CreateRoomEventMessage = "create_room"
	JoinRoomEventMessage   = "join_room"
)

type ErrorPayload struct {
	Reason string `json:"reason"`
}

type CreateRoomPayload struct {
	GameState string `json:"gameState"`
}

type JoinRoomPayload struct {
	RoomID string `json:"room_id"`
}

type TestPayload struct {
	Key string `json:"key"`
}

type EventHandler = func(e Event, c *Client) error

// TODO create event factory
func NewEvent(evtType string, payload any) (Event, error) {
	b, err := json.Marshal(payload)

	if err != nil {
		return Event{}, err
	}

	evt := Event{
		Type:    evtType,
		Payload: b,
	}

	return evt, nil
}

func NewErrorEvent(reason string) (Event, error) {
	p := ErrorPayload{
		Reason: reason,
	}

	b, err := json.Marshal(p)

	if err != nil {
		return Event{}, err
	}

	errEvt := Event{
		Type:    ErrorEventMessage,
		Payload: b,
	}

	return errEvt, nil
}
