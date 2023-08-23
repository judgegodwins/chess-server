package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slices"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type Client struct {
	ID          string
	connection  *websocket.Conn
	manager     *Manager
	egress      chan Event
	JoinedRooms []string
	Data        map[string]interface{}
	err         chan error
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		ID:          uuid.NewString(),
		connection:  conn,
		manager:     manager,
		egress:      make(chan Event),
		JoinedRooms: []string{},
		Data:        make(map[string]interface{}),
		err:         make(chan error),
	}
}

// Reads incoming messages from the clients websocket connection
func (c *Client) readMessages(ctx context.Context) {
	c.connection.SetReadLimit(512)

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.handleError(err)
		return
	}

	c.connection.SetPongHandler(c.pongHandler)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, payload, err := c.connection.ReadMessage()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error reading message: %v", err)
				}
				c.handleError(err)
				return
			}

			var evt Event

			if err := json.Unmarshal(payload, &evt); err != nil {
				c.handleError(err)
				return
			}

			log.Println("event with traceId", evt)

			if err := c.manager.routeEvent(ctx, evt, c); err != nil {
				log.Printf("error handling event %v: %v", evt, err)

				errEvent, err := NewErrorEvent(evt.TraceID, err.Error())

				if err != nil {
					c.handleError(err)
					return
				}

				c.PushToEgress(errEvent)
				// emit an error to client. Any errors returned from event handlers
				// should be emitted to the client using the trace id
			}
		}

	}
}

// writes messages pushed to the client's egress channel
func (c *Client) writeMessages(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		// if the context is cancelled, return
		case <-ctx.Done():
			return
		case message, ok := <-c.egress:
			if !ok { // if client egress conn is closed unexpectedly
				c.handleError(errors.New("client egress channel unexpectedly closed"))
				return
			}

			data, err := json.Marshal(message)

			if err != nil {
				c.handleError(err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				c.handleError(err)
				return
			}
		case <-ticker.C:
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				c.handleError(err)
				return
			}
		}
	}
}

// Sets a new read deadline when a pong is received for a ping message.
func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

// Push error to client error channel. This is used by the
// http handler to know when an error has occurred in a client's readMessage or writeMessage goroutine.
// The http handler closes the connection and removes the client when an error is pushed to the channel
func (c *Client) handleError(e error) {
	c.err <- e
}

// Returns the error channel
func (c *Client) Err() chan error {
	return c.err
}

// Creates an event and pushes to client's egress
func (c *Client) PushEventToEgress(evtType string, payload any) error {
	evt, err := NewEvent(evtType, payload)
	if err != nil {
		return err
	}
	c.PushToEgress(evt)
	return nil
}

// Pushes an event to the client's egress to the delivered via the websocket connection
func (c *Client) PushToEgress(evt Event) {
	c.egress <- evt
}

// Helper method to join a room
func (c *Client) Join(roomId string) {
	c.manager.Lock()
	defer c.manager.Unlock()

	room, ok := c.manager.Rooms[roomId]

	// if room doesn't exist, create one
	if !ok {
		c.manager.Rooms[roomId] = make([]*Client, 0)
		// assign room to newly created room
		room = c.manager.Rooms[roomId]
	}

	// if client is not in room
	if !slices.Contains(room, c) {
		c.manager.Rooms[roomId] = append(room, c) // add client to room
	}

	// if room is not in list of joined rooms
	if !slices.Contains(c.JoinedRooms, roomId) {
		c.JoinedRooms = append(c.JoinedRooms, roomId)
	}
}

// Leave causes a client to leave a room
func (c *Client) Leave(roomId string) {
	c.manager.Lock()
	defer c.manager.Unlock()

	room, ok := c.manager.Rooms[roomId]

	if !ok {
		return
	}

	index := slices.Index(room, c)

	joinedRoomsIndex := slices.Index(c.JoinedRooms, roomId)

	// remove client from room slice
	if index >= 0 {
		c.manager.Rooms[roomId] = append(room[:index], room[index+1:]...)
	}

	// remove roomId from least of joined rooms
	if joinedRoomsIndex >= 0 {
		c.JoinedRooms = append(c.JoinedRooms[:joinedRoomsIndex], c.JoinedRooms[joinedRoomsIndex+1:]...)
	}
}

func (c *Client) LeaveAllRooms() {
	for _, room := range c.JoinedRooms {
		c.Leave(room)
	}
}

// Emits a user_disconnect event to all rooms, a user disconnecting user is part of
func (c *Client) EmitDisconnect() error {
	userID, ok := c.Data["userID"].(string)
	if !ok {
		return errors.New("userID could not be casted to a string")
	}

	userClients := c.manager.Rooms[userID]

	// user still has other clients connected
	if len(userClients) > 1 {
		return nil
	}

	evt, err := NewEvent(EventUserDisconnect, PayloadUserDisconnect{
		UserID: userID,
	})

	if err != nil {
		return err
	}

	for _, room := range c.JoinedRooms {
		c.manager.EmitToRoom(room, evt)
	}

	return nil
}
