package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"

	"context"

	"github.com/go-redis/redis/v8"
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
var redisAddr = "192.168.0.10:6379"

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	ctx := context.Background()
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(pong)
	//
	//	pubsub := redisClient.Subscribe(ctx, "chat")
	//
	//	_, err = pubsub.Receive(ctx)
	//	if err != nil {
	//		panic(err)
	//	}
	//	ch := pubsub.Channel()
	//
	//	err = redisClient.Publish(ctx, "chat", "hello world").Err()
	//	if err != nil {
	//		panic(err)
	//	}
	//	time.AfterFunc(time.Second, func() {
	//		_ = pubsub.Close()
	//	})
	//
	//	for msg := range ch {
	//		fmt.Println(msg.Channel, msg.Payload)
	//	}
	//
	chat := NewChat(redisClient)
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

type Room struct {
	id     string
	users  map[string]*User
	pubsub *redis.PubSub
}

type Chat struct {
	rooms       map[string]*Room
	reg         chan *User
	unreg       chan *User
	broadcast   chan Message
	redisClient *redis.Client
}

func NewChat(rc *redis.Client) *Chat {
	return &Chat{
		rooms:       make(map[string]*Room),
		reg:         make(chan *User, 0),
		unreg:       make(chan *User, 0),
		broadcast:   make(chan Message, 0),
		redisClient: rc,
	}
}

func (chat *Chat) AddRoom(id string) error {
	if _, ok := chat.rooms[id]; ok {
		//TODO the room already exist
		return nil
	}
	ctx := context.Background()
	pubsub := chat.redisClient.Subscribe(ctx, id)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return err
	}
	r := &Room{
		id:     id,
		users:  make(map[string]*User),
		pubsub: pubsub,
	}
	chat.rooms[id] = r
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var m Message
			err := json.Unmarshal([]byte(msg.Payload), &m)
			if err != nil {
				fmt.Println(err)
				break
			}
			for _, u := range r.users {
				u.Ch <- m
			}
		}
	}()
	return nil
}

func (chat *Chat) RemRoom(id string) {
	if _, ok := chat.rooms[id]; ok {
		chat.rooms[id].pubsub.Close()
		delete(chat.rooms, id)
	}
}

func (chat *Chat) Run() {
	//Default room number
	err := chat.AddRoom("default")
	if err != nil {
		panic(err)
	}
	for {
		select {
		case user := <-chat.reg:
			// TODO add room if user entered with other room num than default
			r := chat.rooms[user.RoomID]
			r.users[user.Name] = user
			// broadcast to pubsub user reg
		case user := <-chat.unreg:
			r := user.RoomID
			if _, ok := chat.rooms[r].users[user.Name]; ok {
				delete(chat.rooms[r].users, user.Name)
				user.Conn.Close()
				if len(chat.rooms[r].users) == 0 {
					chat.RemRoom(r)
				}
				// broadcast to pubsub user unreg
			}
			//		case msg := <-chat.broadcast:
			//			for _, u := range chat.users {
			//				fmt.Println("user sent message: ", msg.Username, msg.Text)
			//				go func() { u.Ch <- msg }()
			//			}
		case msg := <-chat.broadcast:
			ctx := context.Background()
			// TODO encode msg to redis format
			err := chat.redisClient.Publish(ctx, msg.RoomID, msg).Err()
			if err != nil {
				panic(err)
			}
			//		case msg:= <-chat.pubsubChan
			//			fmt.Println(msg.Channel, msg.Payload)
			//			for _, u := range chat.users {
			//				go func() { u.Ch <- msg }()
			//			}
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
			RoomID: "default",
			Name:   uname,
			Ch:     make(chan Message, 0),
			Cht:    chat,
			Conn:   conn,
		}
		// TODO check map if user with such name exists
		// chat.rooms[user.RoomID].users[user.Name] = user
		chat.reg <- user
		fmt.Println("user added: ", user)
		go user.read()
		go user.write()
	}
}
