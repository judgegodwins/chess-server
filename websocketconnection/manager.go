package websocketconnection

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/judgegodwins/chess-server/token"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type ClientMap map[string]*Client

type Manager struct {
	tokenMaker token.Maker
	clients    map[string]ClientMap
	handlers map[string]EventHandler
	sync.RWMutex
}

func NewManager(maker token.Maker) *Manager {
	m := &Manager{
		clients:    make(map[string]ClientMap),
		tokenMaker: maker,
		handlers: make(map[string]EventHandler),
	}

	m.setupEventHandlers()
	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[CreateRoomEventMessage] = CreateRoomHandler
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

	if _, ok := m.clients[client.ID]; ok {
		m.clients[client.ID][client.SocketID] = client
	} else {
		m.clients[client.ID] = make(ClientMap)
		m.clients[client.ID][client.SocketID] = client
	}
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client.ID]; ok {
		if _, ok := m.clients[client.ID][client.SocketID]; ok {
			client.connection.Close()
			delete(m.clients[client.ID], client.SocketID)

			log.Println(m.clients)
		}
	}

	log.Println("deleting client")
	log.Println(m.clients)
}

type baseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type dataResponse struct {
	Data interface{} `json:"data"`
	baseResponse
}

func (m *Manager) TokenVerifier(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	header := r.Header.Get("authorization")
	s := strings.Split(header, " ")

	var errResponse baseResponse = baseResponse{
		Success: false,
	}

	if len(s) < 2 {
		w.WriteHeader(http.StatusUnauthorized)
		errResponse.Message = "Invalid authorization header"
		res, err := json.Marshal(errResponse)

		if err != nil {
			log.Println(err)
			return
		}

		w.Write(res)
		return
	}

	payload, err := m.tokenMaker.VerifyToken(s[1])

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		errResponse.Message = err.Error()
		res, err := json.Marshal(errResponse)

		if err != nil {
			log.Println(err)
			return
		}

		w.Write(res)
		return
	}

	response := dataResponse{
		baseResponse: baseResponse{
			Success: true,
			Message: "Temporary auth data",
		},
		Data: map[string]string{
			"id":       payload.ID.String(),
			"username": payload.Username,
		},
	}

	payloadBytes, err := json.Marshal(response)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.Write(payloadBytes)
}

func (m *Manager) TokenHandler(w http.ResponseWriter, r *http.Request) {
	type userTokenRequest struct {
		Username string `json:"username"`
	}

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
		response := baseResponse{
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

	token, payload, err := m.tokenMaker.CreateToken(data.Username, 24*time.Hour)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := dataResponse{
		baseResponse: baseResponse{
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
	log.Println("new socket request")
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
