package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/judgegodwins/chess-server/tokens"
)

type contextkey string

const authContextKey contextkey = "auth_payload"

func (s *Server) AuthMiddleware(c *gin.Context) {
	header := c.Request.Header.Get("authorization")

	if header == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("unauthorized"))
		c.Abort()
		return
	}

	sArr := strings.Split(header, " ")

	if len(sArr) < 2 {
		c.JSON(http.StatusUnauthorized, errorResponse("unauthorized"))
		c.Abort()
		return
	}

	payload, err := tokens.ParseJWTToken(sArr[1], []byte(s.config.JWTSecret))

	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("invalid bearer token"))
		c.Abort()
		return
	}

	c.Set(string(authContextKey), payload)

	c.Next()
}
