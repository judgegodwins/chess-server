package websocketconnection

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/judgegodwins/chess-server/http_utils"
	"github.com/judgegodwins/chess-server/token"
)

type contextkey string

const AuthContextKey contextkey = "auth_payload"

func GetPayload(ctx context.Context) (*token.Payload, bool) {
	payload, ok := ctx.Value(AuthContextKey).(*token.Payload)
	return payload, ok
}

func (m *Manager) AuthMiddleWare(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		header := r.Header.Get("authorization")
		s := strings.Split(header, " ")

		var errResponse http_utils.BaseResponse = http_utils.BaseResponse{
			Success: false,
		}

		if len(s) < 2 {
			w.WriteHeader(http.StatusUnauthorized)
			errResponse.Message = "Invalid authorization header"
			res, err := json.Marshal(errResponse)

			if err != nil {
				panic(err)
			}

			w.Write(res)
			return
		}

		payload, err := m.tokenMaker.VerifyToken(s[1])

		if err != nil {
			errResponse.Message = err.Error()
			res, err := json.Marshal(errResponse)

			if err != nil {
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				panic(err)
			}

			w.WriteHeader(http.StatusUnauthorized)
			w.Write(res)
			return
		}

		ctx := context.WithValue(r.Context(), AuthContextKey, payload)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *Manager) TokenVerifier(w http.ResponseWriter, r *http.Request) {
	payload, ok := GetPayload(r.Context())

	if !ok {
		http.Error(w, "Could not verify authentication", http.StatusInternalServerError)
		log.Println(errors.New("value in auth_payload key of request context could not be casted to *token.Payload"))
		return
	}

	response := http_utils.DataResponse{
		BaseResponse: http_utils.BaseResponse{
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
		return
	}

	w.Write(payloadBytes)
}

func (m *Manager) CreateRoom(w http.ResponseWriter, r *http.Request) {
	payload, ok := r.Context().Value(AuthContextKey).(*token.Payload)

	if !ok {
		http.Error(w, errors.New("authorization error").Error(), http.StatusUnauthorized)
		return
	}

	room := NewRoom(payload.ID.String())

	m.rooms[room.ID] = room

	data, err := json.Marshal(http_utils.DataResponse{
		BaseResponse: http_utils.BaseResponse{
			Success: true,
			Message: "Room created",
		},
		Data: room,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

type verifyRoomRequest struct {
	id string `validate:"required"`
}

func (m *Manager) VerifyRoom(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	req := verifyRoomRequest{
		id: id,
	}

	vErr := http_utils.ValidateStruct(w, m.validate, req)

	if !reflect.ValueOf(vErr).IsZero() {
		http_utils.SendResponse(w, http.StatusBadRequest, vErr)
		return
	}

	room := m.rooms[req.id]
	
	if room == nil {
		http_utils.SendResponse(w, http.StatusBadRequest, http_utils.NewBaseResponse(false, "Invalid room"))
		return
	}

	// TODO check if number of clients is full
	if len(room.Clients) >= 2 {
		http_utils.SendResponse(w, http.StatusBadRequest, http_utils.NewBaseResponse(false, "Game room is fully occupied"))
		return
	}

	http_utils.SendResponse(w, http.StatusOK, http_utils.NewBaseResponse(true, "Room ID is valid"))
}
