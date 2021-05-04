package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"context"

	"github.com/go-redis/redis/v8"
	"github.com/useralexey2000/go-websocket-simple-chat/internal/app/chat"
	"github.com/useralexey2000/go-websocket-simple-chat/internal/pkg/brocker"
)

var redisAddr = "192.168.0.10:6379"

// port for chat server
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
	brk := brocker.NewRedisBrocker(redisClient)
	cht := chat.NewChat(brk)
	go cht.Run()
	fmt.Println("Server started")
	http.HandleFunc("/", chat.RegHandler)
	http.HandleFunc("/index", chat.AuthMiddleware(chat.IndexHandler))
	http.HandleFunc("/ws", chat.AuthMiddleware(chat.WsHandler(cht)))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
