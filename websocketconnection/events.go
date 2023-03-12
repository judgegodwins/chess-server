package websocketconnection

import "encoding/json"

type Event struct {
	Type string `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	ErrorEventMessage = "error"
	CreateRoomEventMessage = "create_room"
)

type ErrorPayload struct {
	Reason string `json:"reason"`
}

type CreateRoomPayload struct {
	GameState string `json:"gameState"`
}

type TestPayload struct {
	Key string `json:"key"`
}

type EventHandler = func(e Event, c *Client) error

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