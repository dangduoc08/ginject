package core

type WS struct {
}

// package main

// import (
// 	"net/http"

// 	"golang.org/x/net/websocket"
// )

// func wsHandler(ws *websocket.Conn) {
// 	defer ws.Close()
// }

// func main() {
// 	s := websocket.Server{
// 		Handler: websocket.Handler(wsHandler),

// 		Handshake: func(cfg *websocket.Config, req *http.Request) error {
// 			// Cho phép mọi Origin
// 			return nil
// 		},
// 	}

// 	http.Handle("/ws", s)

// 	http.ListenAndServe(":4000", nil)
// }
