package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/useralexey2000/go-websocket-simple-chat/internal/pkg/brocker"
)

// Room .
type Room struct {
	ID        string
	users     map[string]*User
	Broadcast chan Message
}

// Chat .
type Chat struct {
	Rooms   map[string]*Room
	reg     chan *User
	unreg   chan *User
	brocker brocker.Brocker
}

// NewChat .
func NewChat(brk brocker.Brocker) *Chat {
	return &Chat{
		Rooms:   make(map[string]*Room),
		reg:     make(chan *User),
		unreg:   make(chan *User),
		brocker: brk,
	}
}

func (chat *Chat) addRoom(id string) error {
	fmt.Printf("adding room num %s\n", id)
	if _, ok := chat.Rooms[id]; ok {
		//TODO the room already exist
		return nil
	}
	ctx := context.Background()
	ch, err := chat.brocker.Sub(ctx, id)
	if err != nil {
		return err
	}
	r := &Room{
		ID:        id,
		users:     make(map[string]*User),
		Broadcast: make(chan Message),
	}
	chat.Rooms[id] = r
	go func() {
		for {
			select {
			// messages from brocker
			case msg := <-ch:
				fmt.Println(msg)
				var m Message
				err := json.Unmarshal([]byte(msg), &m)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("brocker got message: %v", m)
				for _, u := range r.users {
					u.Ch <- m
				}
			// messages to brocker
			case msg := <-r.Broadcast:
				fmt.Println(msg)
				ctx = context.Background()
				m, err := json.Marshal(msg)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("brocker send message: %v", m)
				err = chat.brocker.Pub(ctx, r.ID, m)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}()
	return nil
}

// RemRoom .
func (chat *Chat) remRoom(id string) {
	if _, ok := chat.Rooms[id]; ok {
		chat.brocker.UnSub(id)
		delete(chat.Rooms, id)
	}
}

// Run .
func (chat *Chat) Run() {
	//Default room number
	err := chat.addRoom("default")
	if err != nil {
		panic(err)
	}
	for {
		select {
		case user := <-chat.reg:
			// TODO add room if user entered with other room num than default
			r := chat.Rooms[user.RoomID]
			// Error when reload page
			r.users[user.Name] = user
			// broadcast to pubsub user reg
			fmt.Printf("user registered: %v", user)
		case user := <-chat.unreg:
			r := user.RoomID
			if _, ok := chat.Rooms[r].users[user.Name]; ok {
				delete(chat.Rooms[r].users, user.Name)
				user.Conn.Close()
				fmt.Printf("user unregistered: %v", user)
				if len(chat.Rooms[r].users) == 0 && r != "default" {
					chat.remRoom(r)
				}
			}
		}
	}
}
