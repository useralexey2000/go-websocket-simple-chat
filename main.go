package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

// Message ...
type Message struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

// User ...
type User struct {
	Name string
	Ch   chan Message
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var secretkey = "myauthkey"
var store = sessions.NewCookieStore([]byte(secretkey), nil)

//Broadcast group.
var users = make(map[string]*User)

func main() {

	http.HandleFunc("/", indexHandeler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/ws", wsHandler)

	log.Fatal(http.ListenAndServe(":9000", nil))
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

func wsHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("cant upgrade conn: ", err)
		return
	}
	defer conn.Close()
	//Get sess cookies 1st request.
	_, _, err = conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}

	sess, err := store.Get(r, "login")
	if err != nil {
		log.Println("cant get sess: ", err)
		return
	}

	uname := sess.Values["username"].(string)

	//Chat user.
	user := &User{
		Name: uname,
		Ch:   make(chan Message),
	}
	//Add user to broadcast.
	users[user.Name] = user
	fmt.Println("user added: ", users)

	go func() {
		for {
			msg := <-user.Ch
			if err = conn.WriteJSON(msg); err != nil {
				log.Println("cant write to con: ", err)
				delete(users, user.Name)
				fmt.Println("user deleted: ", users)

				return
			}
		}
	}()
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("cant read from con: ", err)
			delete(users, user.Name)
			fmt.Println("user deleted: ", users)

			return
		}

		msg.Username = user.Name
		//Notify all.
		broadcast(msg)

	}
}

func broadcast(msg Message) {
	for _, u := range users {
		u.Ch <- msg
	}
}
