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

	c.manager.Lock()
	defer c.manager.Unlock()

	room, ok := c.manager.rooms[payload.RoomID]

	if !ok {
		return errors.New("room not found")
	}

	for i, client := range room.Clients {
		// if player logs in through another socket client, maybe in a new browser tab
		if client.ID == c.ID {
			// replace socket client
			room.Clients[i] = c
			event, err := NewEvent("joined_room", room)

			if err != nil {
				return err
			}

			// inform first client it has been disconnected, because
			// player logged in another place
			roomExitEvent, err := NewEvent("conn_elsewhere", nil)
			if err != nil {
				return err
			}

			// inform new client of join
			c.PushToEgress(event)

			// inform old client that player has joined elsewhere
			client.PushToEgress(roomExitEvent)

			return nil
		}
	}

	// if joining client is room creator, grant access immediately
	if room.Creator == c.ID {
		room.Join(c)

		log.Println(room.Clients)
		event, err := NewEvent("joined_room", room)

		if err != nil {
			return err
		}

		c.PushToEgress(event)
	} else {
		// notify creator that a player wants to join
		event, err := NewEvent("requested_join", c)
		if err != nil {
			return err
		}

		creator := room.GetCreator()
		if creator == nil {
			return errors.New("Cannot join room. Creator is not active")
		}

		// send event to creator
		creator.PushToEgress(event)
	}

	return nil
}

func AcceptRequestToJoinHandler(e Event, c *Client) error {
	var payload AcceptJoinRequestPayload

	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		log.Println("error", err, payload)
		return err
	}

	c.manager.Lock()
	defer c.manager.Unlock()

	room, ok := c.manager.rooms[payload.RoomID]

	if !ok {
		return errors.New("room not found")
	}

	clientToJoin, ok := c.manager.clients[payload.PlayerID]

	if !ok || clientToJoin == nil {
		return errors.New("client not found")
	}

	room.Join(clientToJoin)

	event, err := NewEvent("start_game", room)

	if err != nil {
		return err
	}

	room.Broadcast(event, nil)

	return nil
}
