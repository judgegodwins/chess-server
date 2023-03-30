package websocketconnection

import (
	"encoding/json"
	"errors"
	"log"
)

func JoinRoomHandler(e Event, c *Client) error {
	var payload JoinRoomPayload

	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		log.Println("error", err, payload)
		return err
	}

	room, ok := c.manager.rooms[payload.RoomID]

	if !ok {
		return errors.New("room not found")
	}

	room.Clients = append(room.Clients, c)

	event, err := NewEvent("joined_room", room)

	if err != nil {
		return err
	}

	c.egress <- event

	return nil
}

// func CreateRoomHandler(e Event, c *Client) error {
// 	var payload CreateRoomPayload

// 	err := json.Unmarshal(e.Payload, &payload)

// 	if err != nil {
// 		log.Println("error", err, payload)
// 		return err
// 	}

// 	room := NewRoom(payload.GameState, c)
// 	log.Println("room", room)
// 	return nil
// }
