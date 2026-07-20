package core

import (
	"io"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/broker2"
	"github.com/dangduoc08/ginject/common"
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

		item, pattern, ok := ws.eventMatcher.Match(topic)
		if !ok {
			replyException(conn, payload.ID, exception.TopicNotFoundException("no handler registered for topic: "+topic))
			return
		}

		c, ok := runWSMiddlewares(conn, ws, pattern, item, payload)
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
			replyException(conn, payload.ID, exception.WSInternalErrorException(err.Error()))
			return
		}
	}

	reply(conn, TypeAck, payload.ID, payload.Topic)
}

func handleUnsubscribe(conn *WSConnection, connmgr *WSConnmgr, payload WSPayload) {
	for _, topic := range payload.Topic {
		if err := connmgr.Unsubscribe(conn.ID, topic); err != nil {
			replyException(conn, payload.ID, exception.WSInternalErrorException(err.Error()))
			return
		}
	}

	reply(conn, TypeAck, payload.ID, payload.Topic)
}

func handlePublish(conn *WSConnection, ws *WS, payload WSPayload) {
	for _, topic := range payload.Topic {
		item, pattern, ok := ws.eventMatcher.Match(topic)
		if !ok {
			replyException(conn, payload.ID, exception.TopicNotFoundException("no handler registered for topic: "+topic))
			return
		}

		if !ws.connmgr.isSubscribed(conn.ID, topic) {
			replyException(conn, payload.ID, exception.NotSubscribedException("must subscribe before publishing to: "+topic))
			return
		}

		if !dispatchWSEvent(conn, ws, pattern, item, payload) {
			return
		}

		if err := ws.connmgr.Broker.Publish(topic, payload.Message); err != nil {
			replyException(conn, payload.ID, exception.WSInternalErrorException(err.Error()))
			return
		}
	}

	reply(conn, TypeAck, payload.ID, payload.Topic)
}

func runCatchChain(conn *WSConnection, ws *WS, c *ctx.WSContext, pattern string, payloadID string, rec any) {
	c.SetSend(func(data any) {
		reply(conn, TypeError, payloadID, data)
	})

	if _, ok := ws.catchFnsByEvent[pattern]; ok {
		c.Event.Emit(pattern, common.CatchEventPayload{Ctx: c, Recovered: rec, Index: 0})
		return
	}

	replyException(conn, payloadID, *common.NormalizeRecovered(rec))
}

func replyException(conn *WSConnection, id string, ex exception.Exception) {
	reply(conn, TypeError, id, ctx.Map{
		"code":    ex.GetCode(),
		"error":   ex.Error(),
		"message": ex.GetMessage(),
	})
}

func runWSMiddlewares(conn *WSConnection, ws *WS, pattern string, item wsevent.WSEventItem, payload WSPayload) (c *ctx.WSContext, ok bool) {
	c = ws.newCtx()

	defer func() {
		if rec := recover(); rec != nil {
			runCatchChain(conn, ws, c, pattern, payload.ID, rec)
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

func dispatchWSEvent(conn *WSConnection, ws *WS, pattern string, item wsevent.WSEventItem, payload WSPayload) (ok bool) {
	c, mwOK := runWSMiddlewares(conn, ws, pattern, item, payload)
	defer ws.releaseCtx(c)

	if !mwOK {
		return false
	}

	defer func() {
		if rec := recover(); rec != nil {
			runCatchChain(conn, ws, c, pattern, payload.ID, rec)
			ok = false
		}
	}()

	data := ws.resolveAndCallHandler(item.Handler, c)

	if aggregations, ok := c.Context().Value(WithValueKey(pattern)).([]*aggregation.Aggregation); ok {
		var aggregatedData any
		totalAggregations := len(aggregations)

		for i := totalAggregations - 1; i >= 0; i-- {
			agg := aggregations[i]

			if !agg.IsMainHandlerCalled {
				reply(conn, TypeEvent, payload.ID, agg.InterceptorData)
				return true
			}

			if i == totalAggregations-1 && len(data) > 0 {
				aggregatedData = data[len(data)-1].Interface()
			}
			agg.SetMainData(aggregatedData)
			aggregatedData = agg.Aggregate()
		}

		reply(conn, TypeEvent, payload.ID, aggregatedData)
		return true
	}

	if len(data) > 0 {
		reply(conn, TypeEvent, payload.ID, data[len(data)-1].Interface())
	}

	return true
}

func reply(conn *WSConnection, t WSPayloadType, id string, message any) {
	conn.TrySend(WSPayload{ID: id, Type: t, Message: message})
}
