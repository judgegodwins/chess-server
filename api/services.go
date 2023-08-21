package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/judgegodwins/chess-server/tokens"
)

type usernameRequest struct {
	Username string `json:"username" binding:"required"`
}

// Generates a token using the username passed as request body
func (s *Server) TokenGenerator(c *gin.Context) {
	var data usernameRequest

	if err := c.ShouldBindJSON(&data); err != nil {
		c.IndentedJSON(http.StatusUnprocessableEntity, errorResponse(err.Error()))
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
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
		return
	}


	fmt.Println(c.Request.Context())

	c.IndentedJSON(http.StatusOK, successResponse("Auth data", gin.H{
		"id":       payload.ID,
		"username": payload.Username,
		"token":    token,
	}))
}

func (s *Server) CreateRoom(c *gin.Context) {
	var authPayload *tokens.Payload

	v, _ := c.Get(string(authContextKey))

	authPayload, ok := v.(*tokens.Payload)

	if !ok {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
		return
	}

	roomID := uuid.NewString()

	data := make(map[string]string)

	data[roomIDKey] = roomID
	data[roomPlayer1Key] = authPayload.ID
	data[roomPlayer2Key] = ""

	for k, v := range data {
		err := s.rdb.HSet(c, fmt.Sprintf("room:%v", roomID), k, v).Err()

		if err != nil {
			log.Println(err)
			c.IndentedJSON(http.StatusInternalServerError, errorResponse(ErrorMessage500))
			return
		}
	}

	c.IndentedJSON(http.StatusCreated, data)
}
