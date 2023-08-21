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

	if _, ok := m.clients[client.ID]; ok {
		client.connection.Close()
		delete(m.clients, client.ID)
	}
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
		c.IndentedJSON(http.StatusInternalServerError, "something went wrong")
		return
	}

	client := NewClient(conn, m)

	client.Data["userID"] = payload.ID

	m.addClient(client)

	ctx, cancel := context.WithCancel(c)

	defer func() {
		cancel()
		m.removeClient(client)
		err := client.connection.WriteMessage(websocket.CloseMessage, nil)

		if !errors.Is(err, websocket.ErrCloseSent) {
			log.Println("Error sending close message:", err)
		}
	}()

	go client.readMessages(ctx)
	go client.writeMessages(ctx)

	err = <-client.Err()

	log.Println("Client error:", err)
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	switch origin {
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}
