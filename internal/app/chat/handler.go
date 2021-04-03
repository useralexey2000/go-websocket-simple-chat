package chat

import (
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
		//Store username to session.
		sess.Values["username"] = uname
		if err := sess.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		fmt.Println(sess.Values)
		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		http.Error(w, "unsupported request method!", http.StatusBadRequest)
	}
}

// IndexHandler .
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := store.Get(r, "login")
	if err != nil {
		log.Println("cant get sess: ", err)
		return
	}
	fmt.Println(sess.Values)
	//Check if user is logged in.
	if _, ok := sess.Values["username"].(string); !ok {
		http.Redirect(w, r, "/", http.StatusNetworkAuthenticationRequired)
		return
	}
	http.ServeFile(w, r, filepath.Join("../../", "web/chat.html"))
}

// WsHandler .
func WsHandler(chat *Chat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("in wsHandler")
		sess, err := store.Get(r, "login")
		if err != nil {
			log.Println("cant get sess: ", err)
			return
		}
		uname := sess.Values["username"].(string)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("cant upgrade conn: ", err)
			return
		}
		user := &User{
			RoomID: "default",
			Name:   uname,
			Ch:     make(chan Message),
			Cht:    chat,
			Conn:   conn,
		}
		chat.reg <- user
		fmt.Println("user added: ", user)
		go user.read()
		go user.write()
	}
}
