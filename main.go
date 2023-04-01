package main

import (
	"log"
	"net/http"

	"github.com/judgegodwins/chess-server/token"
	"github.com/judgegodwins/chess-server/websocketconnection"
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()
	maker, err := token.NewPasetoMaker("YELLOW SUBMARINE, BLACK WIZARDRY")

	if err != nil {
		log.Fatal(err)
	}

	manager := websocketconnection.NewManager(maker)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods: []string{"POST", "OPTIONS", "GET"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	log.Println(manager)
	// })

	mux.Handle("/", http.FileServer(http.Dir("./frontend")))
	mux.HandleFunc("/ws", manager.ServeWS)
	mux.HandleFunc("/token", manager.TokenHandler)
	mux.Handle("/token/verify", manager.AuthMiddleWare(http.HandlerFunc(manager.TokenVerifier)))
	mux.Handle("/rooms", manager.AuthMiddleWare(http.HandlerFunc(manager.CreateRoom)))

	handler := c.Handler(mux)
	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
