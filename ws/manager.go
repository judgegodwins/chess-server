package ws

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/judgegodwins/chess-server/tokens"
	"github.com/judgegodwins/chess-server/util"
	"github.com/redis/go-redis/v9"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

type ClientList map[string]*Client

type wsQuery struct {
	Token string `form:"token" binding:"required"`
}

type Manager struct {
	clients ClientList
	sync.RWMutex
	handlers map[string]EventHandler
	Rooms    map[string][]*Client
	config   *util.Config
	rdb      *redis.Client
}

func NewManager(config *util.Config, rdb *redis.Client) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		handlers: make(map[string]EventHandler),
		Rooms:    make(map[string][]*Client),
		config:   config,
		rdb:      rdb,
	}

	m.setupEventHandlers()

	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventJoinRoom] = JoinGameRoom
	m.handlers[EventAcceptJoin] = AcceptJoinRequest
	m.handlers[EventPieceMove] = PieceMoveHandler
	m.handlers[EventCloseRoom] = CloseRoom
}

func (m *Manager) routeEvent(ctx context.Context, evt Event, c *Client) error {
	if handler, ok := m.handlers[evt.Type]; ok {
		if err := handler(ctx, evt, c); err != nil {
			return err
		}

		return nil
	}

	return errors.New("there is no such event type")
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client.ID] = client
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	delete(m.clients, client.ID)
}

// Websocket connection handler
func (m *Manager) ServeWS(c *gin.Context) {
	// temporary: use request query to pass token
	var query wsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "token not sent",
		})
		return
	}

	payload, err := tokens.ParseJWTToken(query.Token, []byte(m.config.JWTSecret))

	if err != nil {
		c.IndentedJSON(http.StatusUnauthorized, "unauthorized")
		return
	}

	conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		log.Printf("error upgrading to websocket connection: %v\n", err)
		// c.IndentedJSON(http.StatusInternalServerError, "something went wrong")
		return
	}

	client := NewClient(conn, m)

	client.Data["userID"] = payload.ID
	client.Data["username"] = payload.Username

	m.addClient(client)

	// make client join its own room
	client.Join(payload.ID)

	ctx, cancel := context.WithCancel(c)

	defer func() {
		client.EmitDisconnect()
		cancel()
		client.LeaveAllRooms()
		m.removeClient(client)

		client.connection.Close()
	}()

	go client.readMessages(ctx)
	go client.writeMessages(ctx)

	err = <-client.Err()

	log.Printf("Client (%v) error: %v", client.ID, err)

	c.AbortWithStatus(http.StatusOK)
}

// Emits an event to a room. Every client in that room receives the event.
func (m *Manager) EmitToRoom(roomID string, evt Event) {
	room, ok := m.Rooms[roomID]

	if !ok {
		return
	}

	for _, client := range room {
		client.PushToEgress(evt)
	}
}

// Checks if a client is in the room
func (m *Manager) ClientInRoom(roomID string, c *Client) bool {
	room, ok := m.Rooms[roomID]

	if !ok {
		return false
	}

	for _, client := range room {
		if client == c {
			return true
		}
	}

	return false
}

// util func to emit user_disconnect
func (m *Manager) EmitUserDisconnect(userId, roomId string) error {
	evt, err := NewEvent(EventUserDisconnect, PayloadUser{
		UserID: userId,
	})
	if err != nil {
		return err
	}

	m.EmitToRoom(roomId, evt)
	return nil
}

func (m *Manager) removeRoom(roomID string) {
	m.Lock()
	defer m.Unlock()

	delete(m.Rooms, roomID)
}

func checkOrigin(r *http.Request) bool {
	// origin := r.Header.Get("Origin")
	// switch origin {
	// case "http://localhost:8080":
	// 	return true
	// default:
	// 	return false
	// }

	return true
}
