package ws

import (
	"fmt"
	"io"

	"golang.org/x/net/websocket"
)

type WSPayloadType string

const (
	TypeConnected   WSPayloadType = "connected"
	TypeSubscribe   WSPayloadType = "subscribe"
	TypeUnsubscribe WSPayloadType = "unsubscribe"
	TypePublish     WSPayloadType = "publish"
	TypeEvent       WSPayloadType = "event"
	TypeAck         WSPayloadType = "ack"
	TypeError       WSPayloadType = "error"
	TypePing        WSPayloadType = "ping"
	TypePong        WSPayloadType = "pong"
)

type WSPayload struct {
	Type    WSPayloadType `json:"type"`
	ID      string        `json:"id"`
	Topic   []string      `json:"topic,omitempty"`
	Message any           `json:"message"`
}

func read(wsConn *websocket.Conn) {
	var data WSPayload
	if err := websocket.JSON.Receive(wsConn, &data); err != nil {
		if err != io.EOF {
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println("this is data", data)
}
