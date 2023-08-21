package main

import (
	"log"

	"github.com/judgegodwins/chess-server/api"
	"github.com/judgegodwins/chess-server/util"
	"github.com/redis/go-redis/v9"
)

func main() {
	util.InitValidator()

	config, err := util.LoadConfig(".")

	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       0,
	})

	server := api.NewServer(config, rdb)

	log.Fatal(server.Start())
}
