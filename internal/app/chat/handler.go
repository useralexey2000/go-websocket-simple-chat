package chat

import (
	//	"context"
	//	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var secretkey = "myauthkey"
var store = sessions.NewCookieStore([]byte(secretkey), nil)

// AuthModdleware .
// Checking user authentication
func AuthMiddleware(next func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := store.Get(r, "login")
		if err != nil {
			log.Println("cant get sess: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, ok := sess.Values["username"].(string)
		if !ok {
			http.Redirect(w, r, "/", http.StatusNetworkAuthenticationRequired)
			return
		}
		next(w, r)
	}
}

// RegHandler .
func RegHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, filepath.Join("../../", "web/index.html"))
	case "POST":
		sess, err := store.Get(r, "login")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "can't parse form err: %v", err)
		}
		uname := r.FormValue("username")
		rname := r.FormValue("roomname")
		//Store username and roomname to session.
		sess.Values["username"] = uname
		sess.Values["roomname"] = rname
		if err := sess.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		fmt.Printf("User entered name: %s, room: %s\n", uname, rname)
		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		http.Error(w, "unsupported request method!", http.StatusBadRequest)
	}
}

// IndexHandler .
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("../../", "web/chat.html"))
}

// WsHandler .
func WsHandler(chat *Chat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := store.Get(r, "login")
		uname := sess.Values["username"].(string)
		rname := sess.Values["roomname"].(string)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("cant upgrade conn: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// push to default room
		if rname == "" {
			rname = "default"
		}
		user := &User{
			RoomID: rname,
			Name:   uname,
			Ch:     make(chan Message),
			Cht:    chat,
			Conn:   conn,
		}
		chat.reg <- user
		go user.read()
		go user.write()
	}
}
