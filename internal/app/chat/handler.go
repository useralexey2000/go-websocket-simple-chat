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

//type userContextKey string

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
		//fmt.Println(sess.Values)
		//Check if user is logged in.
		_, ok := sess.Values["username"].(string)
		if !ok {
			http.Redirect(w, r, "/", http.StatusNetworkAuthenticationRequired)
			return
		}
//		userKey := userContextKey("username")
//		ctx := context.WithValue(context.Background(), userKey, u)
//		req := r.WithContext(ctx)
		//fmt.Println("req : ", req.Context().Value(userContextKey("username")))
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
		rname:= r.FormValue("roomname")
		//Store username to session.
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
	//	sess, err := store.Get(r, "login")
	//	if err != nil {
	//		log.Println("cant get sess: ", err)
	//		return
	//	}
	//	fmt.Println(sess.Values)
	//	//Check if user is logged in.
	//	if _, ok := sess.Values["username"].(string); !ok {
	//		http.Redirect(w, r, "/", http.StatusNetworkAuthenticationRequired)
	//		return
	//	}
	http.ServeFile(w, r, filepath.Join("../../", "web/chat.html"))
}

// WsHandler .
func WsHandler(chat *Chat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := store.Get(r, "login")
		uname := sess.Values["username"].(string)
		rname := sess.Values["roomname"].(string)
		// fmt.Println("in wsHandler")
		//v := r.Context().Value(userContextKey("username"))
		//fmt.Println("value is: ", v)
		//if v == nil {
			// TODO check errors dont work
		//	http.Error(w, errors.New("can't get username").Error(), http.StatusInternalServerError)
		//	return
		//}
		// fmt.Println("before checking")s
		//uname, ok := v.(string)
		//if !ok {
			// TODO check errors dont work
		//	http.Error(w, errors.New("can't use username").Error(), http.StatusInternalServerError)
		//	return
		//}
		// fmt.Println("username: ", uname)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("cant upgrade conn: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
