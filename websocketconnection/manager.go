package websocketconnection

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"github.com/judgegodwins/chess-server/http_utils"
	"github.com/judgegodwins/chess-server/token"
	"github.com/samber/lo"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	tokenMaker token.Maker
	validate   *validator.Validate
	clients    map[string]*Client
	rooms      map[string]*Room
	handlers   map[string]EventHandler
	sync.RWMutex
}

func NewManager(maker token.Maker) *Manager {
	m := &Manager{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]*Room),
		tokenMaker: maker,
		handlers:   make(map[string]EventHandler),
		validate:   validator.New(),
	}

	m.setupEventHandlers()
	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[JoinRoomEventMessage] = JoinRoomHandler
	// m.handlers[CreateRoomEventMessage] = CreateRoomHandler
}

func (m *Manager) routeEvents(e Event, c *Client) error {
	if handler, ok := m.handlers[e.Type]; ok {
		if err := handler(e, c); err != nil {
			return err
		}
		return nil
	}
	return errors.New("cannot handle this event")
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()
	m.clients[client.SocketID] = client
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client.SocketID]; ok {
		client.connection.Close()
		delete(m.clients, client.SocketID)
	}

	log.Println("deleting client")
	log.Println(m.clients)
}

type userTokenRequest struct {
	Username string `json:"username" validate:"required"`
}

func (m *Manager) TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	// w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	// w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	var data userTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		response := http_utils.BaseResponse{
			Success: false,
			Message: "invalid body, username required",
		}

		res, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	if err := m.validate.Struct(data); err != nil {
		response := http_utils.ValidationErrorResponse{
			BaseResponse: http_utils.BaseResponse{
				Success: false,
				Message: "invalid body, validation failed",
			},
			Errors: lo.Map(err.(validator.ValidationErrors), func(item validator.FieldError, index int) string {
				return item.Error()
			}),
		}

		res, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	token, payload, err := m.tokenMaker.CreateToken(data.Username, 24*time.Hour)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := http_utils.DataResponse{
		BaseResponse: http_utils.BaseResponse{
			Success: true,
			Message: "token created",
		},
		Data: map[string]string{
			"id":       payload.ID.String(),
			"username": payload.Username,
			"token":    token,
		},
	}

	res, err := json.Marshal(response)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	payload, err := m.tokenMaker.VerifyToken(token)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(payload.ID.String(), payload.Username, conn, m)
	m.addClient(client)

	log.Println("New client", client)

	log.Println("clients", m.clients)

	go client.readMessages()
	go client.writeMessages()
	go client.listenForErrors()
}

func checkOrigin(r *http.Request) bool {
	allowed := map[string]bool{
		"http://localhost:3000": true,
		"http://localhost:3001": true,
	}

	if value := allowed[r.Header.Get("Origin")]; !value {
		return false
	}

	return true
}
