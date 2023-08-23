package api

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/judgegodwins/chess-server/util"
	"github.com/judgegodwins/chess-server/ws"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	config    *util.Config
	wsManager *ws.Manager
	router    *gin.Engine
	rdb       *redis.Client
}

func NewServer(config *util.Config, rdb *redis.Client) *Server {
	router := gin.Default()

	server := &Server{
		config:    config,
		wsManager: ws.NewManager(config, rdb),
		router:    router,
		rdb:       rdb,
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	router.Any("/ws", server.wsManager.ServeWS)
	router.StaticFS("/frontend", http.Dir("./frontend"))
	router.POST("/token", server.TokenGenerator)
	router.POST("/token/verify", server.AuthMiddleware, server.GetTokenData)
	router.POST("/rooms", server.AuthMiddleware, server.CreateRoom)
	router.GET("/rooms/:id", server.AuthMiddleware, server.CheckRoom)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorResponse("endpoint not found"))
	})

	return server
}

func (s *Server) Start() error {
	return s.router.Run(fmt.Sprintf(":%v", s.config.Port))
}
