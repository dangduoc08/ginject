package core

import (
	"fmt"
	"io"

	"github.com/dangduoc08/ginject/broker2"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/wsevent"
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

func readLoop(conn *WSConnection, ws *WS) {
	for {
		var payload WSPayload
		if err := websocket.JSON.Receive(conn.Conn, &payload); err != nil {
			if err != io.EOF {
				ws.logger.Error("WSReadFailed", "error", err)
			}
			return
		}

		ws.connmgr.touch(conn.ID)

		switch payload.Type {
		case TypeSubscribe:
			handleSubscribe(conn, ws, payload)
		case TypeUnsubscribe:
			handleUnsubscribe(conn, ws.connmgr, payload)
		case TypePublish:
			handlePublish(conn, ws, payload)
		case TypePong:
			// liveness only; ws.connmgr.touch above already recorded it.
		default:
			reply(conn, TypeError, payload.ID, "unsupported type: "+string(payload.Type))
		}
	}
}

func handleSubscribe(conn *WSConnection, ws *WS, payload WSPayload) {
	for _, topic := range payload.Topic {
		if ws.connmgr.isSubscribed(conn.ID, topic) {
			continue
		}

		item, _, ok := ws.eventMatcher.Match(topic)
		if !ok {
			reply(conn, TypeError, payload.ID, "no handler registered for topic: "+topic)
			return
		}

		c, ok := runWSMiddlewares(conn, ws, item, payload)
		ws.releaseCtx(c)
		if !ok {
			return
		}

		err := ws.connmgr.Subscribe(conn.ID, topic, func(m *broker2.Message) {
			conn.TrySend(WSPayload{
				Type:    TypeEvent,
				Topic:   []string{m.Topic},
				Message: m.Payload,
			})
		})
		if err != nil {
			reply(conn, TypeError, payload.ID, err.Error())
			return
		}
	}

	reply(conn, TypeAck, payload.ID, payload.Topic)
}

func handleUnsubscribe(conn *WSConnection, connmgr *WSConnmgr, payload WSPayload) {
	for _, topic := range payload.Topic {
		if err := connmgr.Unsubscribe(conn.ID, topic); err != nil {
			reply(conn, TypeError, payload.ID, err.Error())
			return
		}
	}

	reply(conn, TypeAck, payload.ID, payload.Topic)
}

func handlePublish(conn *WSConnection, ws *WS, payload WSPayload) {
	for _, topic := range payload.Topic {
		item, _, ok := ws.eventMatcher.Match(topic)
		if !ok {
			reply(conn, TypeError, payload.ID, "no handler registered for topic: "+topic)
			return
		}

		if !ws.connmgr.isSubscribed(conn.ID, topic) {
			reply(conn, TypeError, payload.ID, "must subscribe before publishing to: "+topic)
			return
		}

		if !dispatchWSEvent(conn, ws, item, payload) {
			return
		}

		if err := ws.connmgr.Broker.Publish(topic, payload.Message); err != nil {
			reply(conn, TypeError, payload.ID, err.Error())
			return
		}
	}

	reply(conn, TypeAck, payload.ID, payload.Topic)
}

func runWSMiddlewares(conn *WSConnection, ws *WS, item wsevent.WSEventItem, payload WSPayload) (c *ctx.WSContext, ok bool) {
	c = ws.newCtx()

	defer func() {
		if rec := recover(); rec != nil {
			msg := fmt.Sprint(rec)
			if ex, isException := rec.(exception.Exception); isException {
				msg = ex.Error()
			}
			reply(conn, TypeError, payload.ID, msg)
			ok = false
		}
	}()

	c.Init(conn.Conn)
	messageMap, _ := payload.Message.(map[string]any)
	c.SetWSPayload(ctx.WSPayload(messageMap))

	isNext := true
	c.Next = func() { isNext = true }

	for _, middleware := range item.Middlewares {
		if isNext {
			isNext = false
			middleware(c)
		}
	}

	return c, isNext
}

func dispatchWSEvent(conn *WSConnection, ws *WS, item wsevent.WSEventItem, payload WSPayload) (ok bool) {
	c, mwOK := runWSMiddlewares(conn, ws, item, payload)
	defer ws.releaseCtx(c)

	if !mwOK {
		return false
	}

	defer func() {
		if rec := recover(); rec != nil {
			msg := fmt.Sprint(rec)
			if ex, isException := rec.(exception.Exception); isException {
				msg = ex.Error()
			}
			reply(conn, TypeError, payload.ID, msg)
			ok = false
		}
	}()

	data := ws.resolveAndCallHandler(item.Handler, c)
	if len(data) > 0 {
		reply(conn, TypeEvent, payload.ID, data[len(data)-1].Interface())
	}

	return true
}

func reply(conn *WSConnection, t WSPayloadType, id string, message any) {
	conn.TrySend(WSPayload{ID: id, Type: t, Message: message})
}
