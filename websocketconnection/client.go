package websocketconnection

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait = 10 * time.Second

	pingInterval = (pongWait * 8) / 10
)

type Client struct {
	ID         string
	Username   string
	manager    *Manager
	connection *websocket.Conn
	egress     chan Event
}

func NewClient(id string, username string, conn *websocket.Conn, m *Manager) *Client {
	return &Client{
		ID:         id,
		Username:   username,
		connection: conn,
		egress:     make(chan Event),
		manager:    m,
	}
}

func (c *Client) readMessages() {
	defer c.manager.removeClient(c)

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}

	c.connection.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			log.Println("socket closure:", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Println("unexpected closure of socket connection:", err)
			}
			break
		}

		var evt Event
		var errEvent Event

		log.Println("event", evt)

		if err := json.Unmarshal(payload, &evt); err != nil {
			errEvent, err = NewErrorEvent("Cannot unmarshal json payload")
			if err != nil {
				log.Println("error creating error event")
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

func (c *Client) writeMessages() {
	defer c.manager.removeClient(c)

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case event, ok := <-c.egress:
			if !ok {
				log.Println("client egress closed")
				if err := c.connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Println("connection closed")
				}
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
			log.Println("pinging")
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("cannot send ping message:", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
