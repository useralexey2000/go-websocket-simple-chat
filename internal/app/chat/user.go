package chat

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// Message .
type Message struct {
	RoomID   string `json:"roomid"`
	Username string `json:"username"`
	Text     string `json:"text"`
}

// User ...
type User struct {
	Name   string
	Ch     chan Message
	RoomID string
	Cht    *Chat
	Conn   *websocket.Conn
}

func (user *User) write() {
	defer user.Conn.Close()
	for {
		msg, ok := <-user.Ch
		if !ok {
			fmt.Printf("Channel of user %s is closed\n", user.Name)
			break
		}
		if err := user.Conn.WriteJSON(msg); err != nil {
			fmt.Println("cant write to con: ", err)
			user.Cht.unreg <- user
			return
		}
	}
}

func (user *User) read() {
	defer user.Conn.Close()
	for {
		var msg Message
		if err := user.Conn.ReadJSON(&msg); err != nil {
			fmt.Println("cant read from conn ", err)
			user.Cht.unreg <- user
			return
		}
		msg.Username = user.Name
		msg.RoomID = user.RoomID
		user.Cht.Rooms[user.RoomID].Broadcast <- msg
	}
}
