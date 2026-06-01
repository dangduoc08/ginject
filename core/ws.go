package core

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"golang.org/x/net/websocket"
)

type WS struct {
	eventToIDMu sync.RWMutex

	eventMap          map[string][]ctx.Handler // to store WS layers, key = subscribe event name
	mainHandlerMap    map[string]any           // to store WS main handler
	eventToID         map[string][]string      // to store WS IDs, key = emit event name
	catchFnsMap       map[string][]common.Catch
	globalMiddlewares *[]common.MiddlewareFn

	invokeHandler func(f any, c *ctx.Context) []reflect.Value
}

func newWS() *WS {
	ws := WS{
		catchFnsMap:    make(map[string][]common.Catch),
		eventMap:       make(map[string][]func(*ctx.Context)),
		mainHandlerMap: make(map[string]any),
		eventToID:      make(map[string][]string),
	}

	return &ws
}

func (ws *WS) handleRequest(wsConn *websocket.Conn, c *ctx.Context) {
	wsInstance := ctx.NewWS(wsConn)
	c.WS = wsInstance
	wsid := wsInstance.GetConnID()
	wsSubscribedEvents := wsInstance.GetSubscribedEvents()
	subprotocol := wsInstance.GetSubprotocol()

	defer func() {
		for _, subscribedEventName := range wsSubscribedEvents {
			ws.removeWSEvent(subscribedEventName, wsid, c)
		}
		_ = wsConn.Close()
	}()

	if !wsInstance.CanEstablish(common.InsertedEvents) {
		return
	}

	for _, subscribedEventName := range wsSubscribedEvents {
		ws.addWSEvent(subscribedEventName, wsid, c, func(args ...any) {
			_ = wsInstance.SendToConn(c, wsConn, args[0].(string))
		})
	}

	for {
		var message []byte
		err := websocket.Message.Receive(wsConn, &message)
		c.Timestamp = time.Now()

		if err != nil {
			if err == io.EOF {
				break
			}
			ws.invokeGlobalMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		var wsMsg ctx.WSMessage
		if err = json.Unmarshal(message, &wsMsg); err != nil {
			ws.invokeGlobalMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		isNext := true
		c.Next = func() { isNext = true }

		if ws.dispatchMessage(c, wsConn, wsMsg, subprotocol, &isNext, wsid, wsSubscribedEvents) {
			return
		}
	}
}

func (ws *WS) dispatchMessage(c *ctx.Context, wsConn *websocket.Conn, wsMsg ctx.WSMessage, subprotocol string, isNext *bool, wsid string, wsSubscribedEvents []string) (recurse bool) {
	publishEventName := common.ToWSEventName(subprotocol, wsMsg.Event)

	defer func() {
		if rec := recover(); rec != nil {
			if errorAggregationOperators, ok := c.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)).([]aggregation.AggregationOperator); ok {
				totalErrorAggregations := len(errorAggregationOperators)

				defer func() {
					if rec := recover(); rec != nil {
						c.Event.Emit(publishEventName, c, rec, 0)
					}
				}()

				for i := totalErrorAggregations - 1; i >= 0; i-- {
					op := errorAggregationOperators[i]
					rec = op(c, rec)
				}
			}

			if _, ok := ws.catchFnsMap[publishEventName]; ok && rec != nil {
				c.Event.Emit(publishEventName, c, rec, 0)
			}

			newCtx := context.WithValue(c.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), nil)
			c.Request = c.WithContext(newCtx)

			for _, eventName := range wsSubscribedEvents {
				ws.removeWSEvent(eventName, wsid, c)
			}

			ws.handleRequest(wsConn, c)
			recurse = true
		}
	}()

	c.WS.Message = wsMsg

	if handlers, isMatched := ws.eventMap[publishEventName]; isMatched {
		for index, handler := range handlers {
			if *isNext {
				*isNext = false
				handler(c)

				if index == len(handlers)-1 && *isNext {
					injectableHandler := ws.mainHandlerMap[publishEventName]

					data := ws.invokeHandler(injectableHandler, c)
					if len(data) == 1 {
						data = append(data, reflect.ValueOf("*"))
						data[1], data[0] = data[0], data[1]
					}
					configPublishedEventName := data[0].String()

					if aggregations, ok := c.Context().Value(WithValueKey(publishEventName)).([]*aggregation.Aggregation); ok {
						var aggregatedData any
						isMainHandlerCalled := true
						totalAggregations := len(aggregations)

						for i := totalAggregations - 1; i >= 0; i-- {
							agg := aggregations[i]

							if agg.IsMainHandlerCalled {
								if i == totalAggregations-1 && len(data) > 1 {
									aggregatedData = data[1].Interface()
								}
								agg.SetMainData(aggregatedData)
								aggregatedData = agg.Aggregate(c)
							} else {
								isMainHandlerCalled = false
								wsMsg2 := toWSMessage(reflect.ValueOf(agg.InterceptorData))
								ws.publishWSEvent(configPublishedEventName, wsMsg2, c)
								break
							}
						}

						if isMainHandlerCalled {
							wsMsg2 := toWSMessage(reflect.ValueOf(aggregatedData))
							ws.publishWSEvent(configPublishedEventName, wsMsg2, c)
						}
					} else {
						if len(data) > 1 {
							wsMsg2 := toWSMessage(data[1])
							ws.publishWSEvent(configPublishedEventName, wsMsg2, c)
						}
					}
				}
			}
		}
	} else {
		ws.invokeGlobalMiddlewares(c, exception.NotFoundException("Cannot emit "+wsMsg.Event+" event"))
	}
	return false
}

func (ws *WS) addWSEvent(subscribedEventName, wsid string, c *ctx.Context, cb func(args ...any)) {
	c.Event.On(subscribedEventName+wsid, cb)
	ws.eventToIDMu.Lock()
	ws.eventToID[subscribedEventName] = append(ws.eventToID[subscribedEventName], wsid)
	ws.eventToIDMu.Unlock()
}

func (ws *WS) removeWSEvent(subscribedEventName, wsid string, c *ctx.Context) {
	c.Event.RemoveAllListeners(subscribedEventName + wsid)
	ws.eventToIDMu.Lock()
	old := ws.eventToID[subscribedEventName]
	filtered := make([]string, 0, len(old))
	for _, id := range old {
		if id != wsid {
			filtered = append(filtered, id)
		}
	}
	ws.eventToID[subscribedEventName] = filtered
	ws.eventToIDMu.Unlock()
}

func (ws *WS) publishWSEvent(configPublishedEventName, wsMsg string, c *ctx.Context) {
	ws.eventToIDMu.RLock()
	wsids := ws.eventToID[configPublishedEventName]
	ws.eventToIDMu.RUnlock()
	for _, wsid := range wsids {
		c.Event.Emit(configPublishedEventName+wsid, wsMsg)
	}
	newCtx := context.WithValue(c.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), nil)
	c.Request = c.WithContext(newCtx)
}

func (ws *WS) invokeGlobalMiddlewares(c *ctx.Context, exception exception.Exception) {
	isNext := true
	c.Next = func() {
		isNext = true
	}

	for _, globalMiddleware := range *ws.globalMiddlewares {
		if isNext {
			isNext = false
			globalMiddleware.Use(c, c.Next)
		}
	}

	if isNext {
		_ = c.WS.SendSelf(c, ctx.Map{
			"code":    exception.GetCode(),
			"error":   exception.Error(),
			"message": exception.GetResponse(),
		})
	}
}
