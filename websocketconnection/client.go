package websocketconnection

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	pongWait = 10 * time.Second

	pingInterval = (pongWait * 8) / 10
)

type Client struct {
	ID         string
	SocketID   string
	Username   string
	manager    *Manager
	connection *websocket.Conn
	egress     chan Event
	readErr    chan error
	writeErr   chan error
	err        chan error
}

func NewClient(id string, username string, conn *websocket.Conn, m *Manager) *Client {
	return &Client{
		ID:         id,
		SocketID:   uuid.NewString(),
		Username:   username,
		connection: conn,
		egress:     make(chan Event),
		readErr:    make(chan error), // client.readMessages listens on this channel for errors that should cause the goroutine to exit
		writeErr:   make(chan error), // client.writeMessages listens on this channel for errors that should cause the goroutine to exit
		err:        make(chan error), // client.listenForErrors 
		manager:    m,
	}
}

// Both readMessage and writeMessage listen to the error channel.
// Pushing to the error channel (c.err) from either readMessages or writeMessages causes the other to exit
// since an error has occured

// Sends an error on the respective error channels for client.writeMessages
// and client.listenForErrors to exit
func (c *Client) readError(err error) {
	c.writeErr <- err
	c.err <- err
}

// Sends an error on the respective error channels for client.readMessages
// and client.listenForErrors to exit
func (c *Client) writeError(err error) {
	c.readErr <- err
	c.err <- err
}

func (c *Client) readMessages() {
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		c.readError(err)
		return
	}

	c.connection.SetPongHandler(c.pongHandler)

	for {
		select {
		case <-c.readErr:
			return
		default:
			_, payload, err := c.connection.ReadMessage()
			if err != nil {
				log.Println("socket closure:", c, err)
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Println("unexpected closure of socket connection:", err)
				}
				c.readError(err)
				return
			}

			var evt Event
			var errEvent Event

			log.Println("event", evt)

			if err := json.Unmarshal(payload, &evt); err != nil {
				errEvent, err = NewErrorEvent("Cannot unmarshal json payload")
				if err != nil {
					log.Println("error creating error event", err)
					continue
				}
				c.egress <- errEvent
				continue
			}

			err = c.manager.routeEvents(evt, c)

			if err != nil {
				errEvent, err = NewErrorEvent(err.Error())
				if err != nil {
					log.Println(err)
					continue
				}

				c.egress <- errEvent
			}
		}
	}
}

func (c *Client) writeMessages() {
	ticker := time.NewTicker(pingInterval)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case event, ok := <-c.egress:
			if !ok {
				log.Println("client egress closed")
				if err := c.connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Println("connection closed")
				}

				c.writeError(fmt.Errorf("egress connection closed for client with id %v", c.ID));
				return
			}

			message, err := json.Marshal(event)

			if err != nil {
				log.Printf("err marshalling event %v to message for client %v", event, c.ID)
				continue
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("write error:", err)
			}
		case <-ticker.C:
			log.Println("ping client", c.manager.clients)
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("cannot send ping message:", err)
				c.writeError(err)
				return
			}
		case err := <-c.writeErr:
			log.Println("error from channel", err)
			return
		}
	}
}

// Listens for errors and closes a connection
func (c *Client) listenForErrors() {
	defer c.manager.removeClient(c)
	<-c.err
	log.Println("caught error about to delete")
}

func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
