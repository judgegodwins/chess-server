package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	// "time"

	"github.com/judgegodwins/chess-server/util"
)

func JoinGameRoom(ctx context.Context, e Event, c *Client) error {
	var payload PayloadRoom

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
		err := c.PushEventToEgress(EventRoomNotFound, nil)
		if err != nil {
			return err
		}
		return nil
	}

	userID, ok := c.Data["userID"].(string)

	if !ok {
		return fmt.Errorf("userID not found in Data map of client with id %v", c.ID)
	}

	fmt.Println("roomplayer1 and userID", room[util.RoomPlayer1Key], userID)

	// if client is already one of the players in the room
	if room[util.RoomPlayer1Key] == userID || room[util.RoomPlayer2Key] == userID {
		// if the connecting user has a tab/device already connected to this room (maybe on some other device)
		// disconnect them from the room on the other device
		if len(c.manager.Rooms[payload.RoomID]) > 0 {
			for _, client := range c.manager.Rooms[payload.RoomID] {
				if client.Data["userID"] == userID && client.ID != c.ID {
					client.PushEventToEgress("conn_elsewhere", payload.RoomID)
				}
			}
		}
		fmt.Println("room Created", payload.RoomID)
		// make client join room
		c.Join(payload.RoomID)

		fmt.Println("manager rooms", c.manager.Rooms)

		// create a joined_room event that'll tell the client that it has joined a room
		err := c.PushEventToEgress("joined_room", room)
		if err != nil {
			return err
		}

		// userClients := c.manager.Rooms[userID]

		// // if user clients is 1, then the user was previously disconnected and just reconnected
		// if len(userClients) == 1 {
		evt, err := NewEvent(EventUserConnect, PayloadUser{
			UserID: userID,
		})

		if err != nil {
			return err
		}

		// emit user_connect to the room to inform the client that the opponent is connected
		c.manager.EmitToRoom(payload.RoomID, evt)

		// if game is already started
		if room[util.RoomGameStartedKey] == util.GameStartedTrue.String() {
			// if user is player1 and player2 is disconnected, tell joining user that the opponent is disconnected
			if room[util.RoomPlayer1Key] == userID {
				if len(c.manager.Rooms[room[util.RoomPlayer2Key]]) == 0 {
					err := c.manager.EmitUserDisconnect(room[util.RoomPlayer2Key], payload.RoomID)

					if err != nil {
						return err
					}
				}
				// else if user is player2 and player1 is disconnected, tell joining user that the opponent is disconnected
			} else if room[util.RoomPlayer2Key] == userID {
				if len(c.manager.Rooms[room[util.RoomPlayer1Key]]) == 0 {
					err := c.manager.EmitUserDisconnect(room[util.RoomPlayer1Key], payload.RoomID)

					if err != nil {
						return err
					}
				}
			}
		}

		return nil
		// exit func
	}

	if room[util.RoomPlayer2Key] != "" {
		err := c.PushEventToEgress("room_full", nil)
		if err != nil {
			return err
		}
		return nil
	}

	evt, err := NewEvent("request_join", map[string]interface{}{
		"id":        c.Data["userID"],
		"client_id": c.ID,
		"username":  c.Data["username"],
	})

	if err != nil {
		return err
	}

	// emit request_join event to room creator
	c.manager.EmitToRoom(payload.RoomID, evt)
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

	client := c.manager.clients[payload.ClientID]

	if client == nil {
		return errors.New("the second player is not online")
	}

	username, ok := client.Data["username"].(string)

	if !ok {
		log.Printf("username for client (%v) not set", client)
		return errors.New("an error occurred while adding the opponent to the room")
	}

	client.Join(payload.RoomID)

	// set player2 to user ID
	if err := c.manager.rdb.HSet(ctx, roomKey, util.RoomPlayer2Key, payload.PlayerID).Err(); err != nil {
		return err
	}

	if err := c.manager.rdb.HSet(ctx, roomKey, util.RoomPlayer2UsernameKey, username).Err(); err != nil {
		return err
	}

	// set active = "yes"
	if err = c.manager.rdb.HSet(ctx, roomKey, util.RoomGameStartedKey, util.GameStartedTrue.String()).Err(); err != nil {
		return err
	}

	room[util.RoomPlayer2Key] = payload.PlayerID
	room[util.RoomPlayer2UsernameKey] = username
	room[util.RoomGameStartedKey] = util.GameStartedTrue.String()

	// create start_game event
	evt, err := NewEvent(EventStartGame, room)
	if err != nil {
		return err
	}

	// emit start_game event to game room
	c.manager.EmitToRoom(payload.RoomID, evt)

	return nil
}

func PieceMoveHandler(ctx context.Context, e Event, c *Client) error {
	var payload PayloadPieceMove

	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return err
	}

	roomKey := util.GetRoomKey(payload.RoomID)

	c.manager.EmitToRoom(payload.RoomID, e)

	// update FEN state of game
	if err := c.manager.rdb.HSet(ctx, roomKey, util.RoomGameStateKey, payload.Fen).Err(); err != nil {
		return err
	}

	return nil
}

func CloseRoom(ctx context.Context, e Event, c *Client) error {
	var payload PayloadRoom

	// unmarshal payload bytes
	err := json.Unmarshal(e.Payload, &payload)

	if err != nil {
		return err
	}

	// delete room data on redis
	if err = c.manager.rdb.Del(ctx, util.GetRoomKey(payload.RoomID)).Err(); err != nil {
		return err
	}

	// create closing_room event
	evt, err := NewEvent(EventClosingRoom, PayloadRoom{
		RoomID: payload.RoomID,
	})

	if err != nil {
		return err
	}

	// emit closing_room event to clients in room
	c.manager.EmitToRoom(payload.RoomID, evt)

	// remove room
	c.manager.removeRoom(payload.RoomID)

	return nil
}
