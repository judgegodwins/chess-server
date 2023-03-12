package websocketconnection

import (
	"encoding/json"
	"log"
)

func CreateRoomHandler(e Event, c *Client) error {
	var payload CreateRoomPayload

	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		log.Println("error", err, payload)
		return err
	}

	room := NewRoom(payload.GameState, c)
	log.Println("room", room)
	return nil
}