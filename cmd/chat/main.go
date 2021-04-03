package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"context"

	"github.com/go-redis/redis/v8"
	"github.com/useralexey2000/go-websocket-simple-chat/internal/app/chat"
)

var redisAddr = "192.168.0.10:6379"
var port string

func init() {
	flag.StringVar(&port, "port", "9000", "Specify tcp port.")
}

func main() {
	flag.Parse()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	ctx := context.Background()
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pong)
	cht := chat.NewChat(redisClient)
	go cht.Run()
	http.HandleFunc("/", chat.RegHandler)
	http.HandleFunc("/index", chat.IndexHandler)
	http.HandleFunc("/ws", chat.WsHandler(cht))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
