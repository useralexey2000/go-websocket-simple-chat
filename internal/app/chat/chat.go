package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/useralexey2000/go-websocket-simple-chat/internal/pkg/brocker"
)

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

// Run chat .
func (chat *Chat) Run() {
	for {
		select {
		case user := <-chat.reg:
			room, ok := chat.Rooms[user.RoomID]
			if !ok {
				rr, err := chat.addRoom(user.RoomID)
				if err != nil {
					panic(err)
				}
				room = rr
				go chat.serveRoom(room)
			}
			// chat.mux.Unlock()
			err := room.addUser(user)
			if err != nil {
				fmt.Println("Cant add user reason: ", err)
				close(user.Ch)
				continue
			}
			// r.users[user.Name] = user
			//TODO  broadcast to pubsub user reg
			fmt.Printf("user %s registered in room: %s\n", user.Name, user.RoomID)
		case user := <-chat.unreg:
			room := chat.Rooms[user.RoomID]
			room.remUser(user)
			chat.checkRoom(room)
		}
	}
}

// add new room .
func (chat *Chat) addRoom(id string) (*Room, error) {
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
		subChan:   ch,
		done:      make(chan struct{}),
	}
	chat.Rooms[id] = r
	return r, nil
}

// remove room .
func (chat *Chat) remRoom(r *Room) {
	if _, ok := chat.Rooms[r.ID]; ok {
		fmt.Printf("Unsubscribing room %s from subscription\n", r.ID)
		chat.brocker.UnSub(r.ID)
		close(r.done)
		delete(chat.Rooms, r.ID)
		fmt.Printf("Room %s deleted from chat server\n", r.ID)
	}
}

// check if there are any users in room, remove if not .
func (chat *Chat) checkRoom(r *Room) {
	if len(r.users) == 0 {
		fmt.Printf("No users in room: %s, removing room\n", r.ID)
		chat.remRoom(r)
		s := make([]string, 0)
		for k := range chat.Rooms {
			s = append(s, k)
		}
		fmt.Println("Remaining rooms: ", s)
	}
}

// serve room .
func (chat *Chat) serveRoom(r *Room) {
	for {
		select {
		// messages from brocker
		case msg := <-r.subChan:
			fmt.Printf("Room %s received message: %s\n", r.ID, msg)
			var m Message
			err := json.Unmarshal([]byte(msg), &m)
			if err != nil {
				fmt.Println(err)
			}
			r.mux.Lock()
			for _, u := range r.users {
				select {
				case u.Ch <- m:
				default:
					close(u.Ch)
				}
			}
			r.mux.Unlock()
			// messages to brocker
		case msg := <-r.Broadcast:
			m, err := json.Marshal(msg)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("brocker sent message: %s", m)
			ctx := context.Background()
			err = chat.brocker.Pub(ctx, r.ID, m)
			if err != nil {
				fmt.Println(err)
			}
		case <-r.done:
			fmt.Println("Received signal to stop serving room: ", r.ID)
			return
		}
	}
}
