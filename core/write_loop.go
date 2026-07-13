package core

import (
	"github.com/dangduoc08/ginject/common"
	"golang.org/x/net/websocket"
)

func writeLoop(wsConn *websocket.Conn, send <-chan WSPayload, done <-chan struct{}, logger common.Logger) {
	for {
		select {
		case payload := <-send:
			if err := websocket.JSON.Send(wsConn, payload); err != nil {
				logger.Error("WSWriteFailed", "error", err)
				return
			}
		case <-done:
			return
		}
	}
}
