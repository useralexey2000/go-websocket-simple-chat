package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// Room .
type Room struct {
	id     string
	users  map[string]*User
	pubsub *redis.PubSub
}

// Chat .
type Chat struct {
	rooms       map[string]*Room
	reg         chan *User
	unreg       chan *User
	broadcast   chan Message
	redisClient *redis.Client
}

// NewChat .
func NewChat(rc *redis.Client) *Chat {
	return &Chat{
		rooms:       make(map[string]*Room),
		reg:         make(chan *User),
		unreg:       make(chan *User),
		broadcast:   make(chan Message),
		redisClient: rc,
	}
}

// AddRoom .
func (chat *Chat) AddRoom(id string) error {
	if _, ok := chat.rooms[id]; ok {
		//TODO the room already exist
		return nil
	}
	ctx := context.Background()
	pubsub := chat.redisClient.Subscribe(ctx, id)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return err
	}
	r := &Room{
		id:     id,
		users:  make(map[string]*User),
		pubsub: pubsub,
	}
	chat.rooms[id] = r
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var m Message
			err := json.Unmarshal([]byte(msg.Payload), &m)
			if err != nil {
				fmt.Println(err)
				break
			}
			for _, u := range r.users {
				u.Ch <- m
			}
		}
	}()
	return nil
}

// RemRoom .
func (chat *Chat) RemRoom(id string) {
	if _, ok := chat.rooms[id]; ok {
		chat.rooms[id].pubsub.Close()
		delete(chat.rooms, id)
	}
}

// Run .
func (chat *Chat) Run() {
	//Default room number
	err := chat.AddRoom("default")
	if err != nil {
		panic(err)
	}
	for {
		select {
		case user := <-chat.reg:
			// TODO add room if user entered with other room num than default
			r := chat.rooms[user.RoomID]
			// Error when reload page
			r.users[user.Name] = user
			// broadcast to pubsub user reg
		case user := <-chat.unreg:
			r := user.RoomID
			if _, ok := chat.rooms[r].users[user.Name]; ok {
				delete(chat.rooms[r].users, user.Name)
				user.Conn.Close()
				if len(chat.rooms[r].users) == 0 {
					chat.RemRoom(r)
				}
				// broadcast to pubsub user unreg
			}
		case msg := <-chat.broadcast:
			ctx := context.Background()
			// TODO encode msg to json format
			encMsg, err := json.Marshal(msg)
			if err != nil {
				panic(err)
			}
			err = chat.redisClient.Publish(ctx, msg.RoomID, encMsg).Err()
			if err != nil {
				panic(err)
			}
		}
	}
}
