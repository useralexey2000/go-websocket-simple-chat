package chat

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// Message .
type Message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
	RoomID   string
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
	for {
		msg := <-user.Ch
		if err := user.Conn.WriteJSON(msg); err != nil {
			fmt.Println("cant write to con: ", err)
			user.Cht.unreg <- user
			return
		}
	}
}

func (user *User) read() {
	for {
		var msg Message
		if err := user.Conn.ReadJSON(&msg); err != nil {
			fmt.Println("cant read from conn ", err)
			user.Cht.unreg <- user
			return
		}
		msg.Username = user.Name
		msg.RoomID = user.RoomID
		user.Cht.broadcast <- msg
	}
}
