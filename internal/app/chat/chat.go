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
	subChan   <-chan string
	done	  chan struct{}
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

func (chat *Chat) addRoom(id string)(*Room, error) {
	fmt.Printf("adding room num %s\n", id)
	ctx := context.Background()
	ch, err := chat.brocker.Sub(ctx, id)
	if err != nil {
		return nil, err
	}
	r := &Room{
		ID:        id,
		users:     make(map[string]*User),
		Broadcast: make(chan Message),
		subChan:    ch,
		done:      make(chan struct{}),
	}
	chat.Rooms[id] = r
	chat.serveRoom(r)
	return r, nil
}

func (chat *Chat) serveRoom(r *Room) {
	go func() {
		for {
			select {
			// messages from brocker
			case msg := <-r.subChan:
				fmt.Printf("Room %s received message: %s", r.ID, msg)
				var m Message
				err := json.Unmarshal([]byte(msg), &m)
				if err != nil {
					fmt.Println(err)
				}
				for _, u := range r.users {
					u.Ch <- m
				}
			// messages to brocker
			case msg := <-r.Broadcast:
				ctx := context.Background()
				m, err := json.Marshal(msg)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("brocker sent message: %s", m)
				err = chat.brocker.Pub(ctx, r.ID, m)
				if err != nil {
					fmt.Println(err)
				}
			case <-r.done:
				fmt.Println("Received signal to stop serving room: ", r.ID)
				return
			}
		}
	}()

}

// RemRoom .
func (chat *Chat) remRoom(id string) {
	if r, ok := chat.Rooms[id]; ok {
		fmt.Printf("Unsubscribing room %s from subscription", id)
		chat.brocker.UnSub(id)
		close(r.done)
		delete(chat.Rooms, id)
		fmt.Printf("Room %s deleted from chat server", id)
	}
}

// Run .
func (chat *Chat) Run() {
	for {
		select {
		case user := <-chat.reg:
			//  add room if user entered with other room num than default
			if user.RoomID == "" {
				user.RoomID = "default"
			}
			r, ok := chat.Rooms[user.RoomID]
			if !ok {
				rr, err := chat.addRoom(user.RoomID)
				if err != nil {
					panic(err)
				}
				r = rr
			}
			r.users[user.Name] = user
			// broadcast to pubsub user reg
			fmt.Printf("user %s registered in room: %s\n", user.Name, user.RoomID)
		case user := <-chat.unreg:
			r := user.RoomID
			if _, ok := chat.Rooms[r].users[user.Name]; ok {
				delete(chat.Rooms[r].users, user.Name)
				user.Conn.Close()
				fmt.Printf("user unregistered: %s\n", user.Name)
				if len(chat.Rooms[r].users) == 0 {
					fmt.Printf("No users in room: %s, removing room\n", r)
					chat.remRoom(r)
					s := make([]string, 0)
					for k, _ := range chat.Rooms {
						s = append(s, k)
					}
					fmt.Println("Remaining rooms: ", s)
				}
			}
		}
	}
}
