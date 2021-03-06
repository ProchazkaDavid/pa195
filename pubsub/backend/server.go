package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func serveWs(pool *Pool, w http.ResponseWriter, r *http.Request) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	return client.listen()
}

func main() {
	fmt.Println("Chat App backend is running on port 8080.")
	fmt.Println("Use a command line argument to specify a limit for postgres.")

	pool := newPool()
	go pool.start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if err := serveWs(pool, w, r); err != nil {
			log.Fatalln(err)
		}
	})

	http.ListenAndServe(":8080", nil)
}
