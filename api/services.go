package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/judgegodwins/chess-server/tokens"
	"github.com/judgegodwins/chess-server/util"
)

type usernameRequest struct {
	Username string `json:"username" binding:"required"`
}

// Generates a token using the username passed as request body
func (s *Server) TokenGenerator(c *gin.Context) {
	var data usernameRequest

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse(err.Error()))
		return
	}

	payload := tokens.Payload{
		ID:       uuid.NewString(),
		Username: data.Username,
	}

	token, err := tokens.NewJWTToken(jwt.MapClaims{
		"username": payload.Username,
		"id":       payload.ID,
	}, []byte(s.config.JWTSecret))

	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
		return
	}

	c.JSON(http.StatusOK, successResponse("Auth data", gin.H{
		"id":       payload.ID,
		"username": payload.Username,
		"token":    token,
	}))
}

func (s *Server) GetTokenData(c *gin.Context) {
	payload, ok := GetPayload(c)

	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
		log.Println(errors.New("value in auth_payload key of request context could not be casted to *token.Payload"))
		return
	}

	c.JSON(http.StatusOK, successResponse("success", payload))
}

func (s *Server) CreateRoom(c *gin.Context) {
	authPayload, ok := GetPayload(c)

	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
		log.Println(errors.New("value in auth_payload key of request context could not be casted to *token.Payload"))
		return
	}

	roomID := uuid.NewString()

	data := make(map[string]string)

	data[util.RoomIDKey] = roomID
	data[util.RoomPlayer1Key] = authPayload.ID
	data[util.RoomPlayer2Key] = ""
	data[util.RoomGameStateKey] = util.DefaultFEN
	data[util.RoomGameStartedKey] = util.GameStartedFalse.String()
	data[util.RoomPlayer1UsernameKey] = authPayload.Username

	roomKey := util.GetRoomKey(roomID)
	for k, v := range data {
		err := s.rdb.HSet(c.Request.Context(), roomKey, k, v).Err()

		if err != nil {
			log.Println("error on rdb.HSet:", err)
			c.JSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
			return
		}
	}

	s.rdb.Expire(c.Request.Context(), roomKey, 12*time.Hour).Err()

	c.JSON(http.StatusCreated, successResponse("Room created", data))
}

type checkRoomRequest struct {
	RoomID string `uri:"id" binding:"required"`
}

func (s *Server) CheckRoom(c *gin.Context) {
	var data checkRoomRequest

	if err := c.ShouldBindUri(&data); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse(err.Error()))
		return
	}

	room, err := s.rdb.HGetAll(c.Request.Context(), util.GetRoomKey(data.RoomID)).Result()

	if err != nil {
		log.Println("error getting room data from redis:", err)
		c.JSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
		return
	}

	if len(room) == 0 {
		c.JSON(http.StatusNotFound, errorResponse("room not found"))
		return
	}

	c.JSON(http.StatusOK, successResponse("room data", gin.H{
		"id":   room["id"],
		"full": room[util.RoomPlayer1Key] != "" && room[util.RoomPlayer2Key] != "",
	}))
}
