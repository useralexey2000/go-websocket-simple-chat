package chat

import (
	"errors"
	"fmt"
	"sync"
)

// Room .
type Room struct {
	ID        string
	mux       sync.Mutex
	users     map[string]*User
	Broadcast chan Message
	subChan   <-chan string
	done      chan struct{}
}

func (r *Room) addUser(u *User) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	_, ok := r.users[u.Name]
	if !ok {
		r.users[u.Name] = u
		return nil
	}
	return errors.New(fmt.Sprintf("User with name  %s exists", u.Name))

}

func (r *Room) remUser(u *User) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if _, ok := r.users[u.Name]; ok {
		close(u.Ch)
		delete(r.users, u.Name)
		fmt.Printf("Removed user %s from room: %s\n", u.Name, u.RoomID)
	}
}
