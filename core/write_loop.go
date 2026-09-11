package core

import (
	"time"

	"github.com/dangduoc08/ginject/common"
	"golang.org/x/net/websocket"
)

func writeLoop(wsConn *websocket.Conn, send <-chan WSPayload, done <-chan struct{}, writeTimeout time.Duration, logger common.Logger) {
	for {
		select {
		case payload := <-send:
			if writeTimeout > 0 {
				_ = wsConn.SetWriteDeadline(time.Now().Add(writeTimeout))
			}
			if err := websocket.JSON.Send(wsConn, payload); err != nil {
				logger.Error("WSWriteFailed", "error", err)
				_ = wsConn.Close()
				return
			}
		case <-done:
			return
		}
	}
}
