package main

import (
	"fmt"
	"log"
	"net/http"

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

func main() {
	chat := NewChat()
	go chat.Run()
	http.HandleFunc("/", indexHandeler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/ws", wsHandler(chat))
	log.Fatal(http.ListenAndServe(":9000", nil))
}

// Message ...
type Message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

// User ...
type User struct {
	Name string
	Ch   chan Message
	Cht  *Chat
	Conn *websocket.Conn
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
		user.Cht.broadcast <- msg
	}
}

type Chat struct {
	users     map[string]*User
	reg       chan *User
	unreg     chan *User
	broadcast chan Message
}

func NewChat() *Chat {
	return &Chat{
		users:     make(map[string]*User),
		reg:       make(chan *User),
		unreg:     make(chan *User),
		broadcast: make(chan Message),
	}
}

func (chat *Chat) Run() {
	for {
		select {
		case user := <-chat.reg:
			chat.users[user.Name] = user
		case user := <-chat.unreg:
			if _, ok := chat.users[user.Name]; ok {
				delete(chat.users, user.Name)
				user.Conn.Close()
			}
		case msg := <-chat.broadcast:
			for _, u := range chat.users {
                fmt.Println("user sent message: ", msg.Username, msg.Text)
				go func() { u.Ch <- msg }()
			}
		}
	}
}

func indexHandeler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "index.html")
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
		http.Redirect(w, r, "/chat", http.StatusFound)
	default:
		http.Error(w, "unsupported request method!", http.StatusBadRequest)
	}
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
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
	http.ServeFile(w, r, "chat.html")
}

func wsHandler(chat *Chat) http.HandlerFunc {
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
			Name: uname,
			Ch:   make(chan Message),
			Cht:  chat,
			Conn: conn,
		}
		// TODO check map if user with such name exists
		chat.users[user.Name] = user
		fmt.Println("user added: ", user)
		go user.read()
		go user.write()
	}
}
