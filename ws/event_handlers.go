package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/judgegodwins/chess-server/util"
)

func JoinGameRoom(ctx context.Context, e Event, c *Client) error {
	var payload PayloadJoinRoom

	// unmarshal payload bytes
	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		return err
	}

	roomKey := util.GetRoomKey(payload.RoomID)

	// get room data from redis
	room, err := c.manager.rdb.HGetAll(ctx, roomKey).Result()

	if err != nil {
		return err
	}

	if len(room) == 0 {
		err := c.PushEventToEgress("room_not_found", nil)
		if err != nil {
			return err
		}
		return nil
	}

	userID, ok := c.Data["userID"].(string)

	if !ok {
		return fmt.Errorf("userID not found in Data map of client with id %v", c.ID)
	}

	// if client is already one of the players in the room
	if room[util.RoomPlayer1Key] == userID || room[util.RoomPlayer2Key] == userID {
		// if the connecting user has a tab/device already connected to this room (maybe on some other device)
		// disconnect them from the room on the other device
		for _, client := range c.manager.Rooms[payload.RoomID] {
			if client.Data["userID"] == userID && client.ID != c.ID {
				client.PushEventToEgress("conn_elsewhere", payload.RoomID)
			}
		}

		// make client join room
		c.Join(payload.RoomID)

		// create a joined_room event that'll tell the client that it has joined a room
		err := c.PushEventToEgress("joined_room", room)

		if err != nil {
			return err
		}

		return nil
		// exit func
	}

	fmt.Println("Room", room)
	fmt.Println("player1ID", room[util.RoomPlayer2Key])

	if room[util.RoomPlayer2Key] != "" {
		err := c.PushEventToEgress("room_full", nil)
		if err != nil {
			return err
		}

		return nil
	}

	evt, err := NewEvent("request_join", map[string]interface{}{
		"id":       c.Data["userID"],
		"username": c.Data["username"],
	})

	if err != nil {
		return err
	}

	// emit request_join event to room creator
	c.manager.EmitToRoom(room[util.RoomPlayer1Key], evt)

	return nil
}

func AcceptJoinRequest(ctx context.Context, e Event, c *Client) error {
	var payload PayloadAcceptJoinRequest

	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		return err
	}

	roomKey := util.GetRoomKey(payload.RoomID)

	room, err := c.manager.rdb.HGetAll(ctx, roomKey).Result()

	if err != nil {
		return err
	}

	player1ID := room[util.RoomPlayer1Key]

	if len(room) == 0 || player1ID == "" {
		return errors.New("room details not found")
	}

	// the joining user's personal room, where all connected clients can be found
	playerRoom := c.manager.Rooms[payload.PlayerID]
	player2Active := len(playerRoom) >= 0

	if !player2Active {
		return errors.New("the second player is not active")
	}

	var username string

	for _, client := range playerRoom {
		// let the requesting user join the room
		client.Join(payload.RoomID)

		if username == "" {
			u, _ := client.Data["username"].(string)
			username = u
		}

		// set player2 to user ID
		if err := c.manager.rdb.HSet(ctx, roomKey, util.RoomPlayer2Key, payload.PlayerID).Err(); err != nil {
			return err
		}

		if err := c.manager.rdb.HSet(ctx, roomKey, util.RoomPlayer2UsernameKey, username).Err(); err != nil {
			return err
		}

		// set game_started = "yes"
		if err = c.manager.rdb.HSet(ctx, roomKey, util.RoomGameStartedKey, util.GameStartedTrue.String()).Err(); err != nil {
			return err
		}

		room[util.RoomPlayer2Key] = payload.PlayerID
		room[util.RoomPlayer2UsernameKey] = username
		room[util.RoomGameStartedKey] = util.GameStartedTrue.String()
	}

	// create start_game event
	evt, err := NewEvent("start_game", room)

	if err != nil {
		return err
	}

	// emit start_game event to game room
	c.manager.EmitToRoom(payload.RoomID, evt)

	return nil
}

func PieceMoveHandler(ctx context.Context, e Event, c *Client) error {
	fmt.Println("event", e)
	var payload PayloadPieceMove

	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		return err
	}

	fmt.Println("piece move payload", payload)

	c.manager.EmitToRoom(payload.RoomID, e)

	return nil
}
