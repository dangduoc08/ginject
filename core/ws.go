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
	isNext := true
	c.Next = func() {
		isNext = true
	}
	wsid := wsInstance.GetConnID()
	wsSubscribedEvents := wsInstance.GetSubscribedEvents()

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

		// listen on comming messages
		var message []byte
		err := websocket.Message.Receive(wsConn, &message)

		// reset timestamp
		// based on time when receive message
		c.Timestamp = time.Now()

		if err != nil {

			// client close connection
			if err == io.EOF {
				break
			}
			ws.invokeGlobalMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		var wsMsg ctx.WSMessage
		err = json.Unmarshal(message, &wsMsg)
		if err != nil {
			ws.invokeGlobalMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		// event was registered by controller
		var publishEventName string
		defer func() {
			if rec := recover(); rec != nil {

				// Pipe errors run first
				// then exception filter
				if errorAggregationOperators, ok := c.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)).([]aggregation.AggregationOperator); ok {
					totalErrorAggregations := len(errorAggregationOperators)

					// Handle case if pipe error panic
					defer func() {
						if rec := recover(); rec != nil {
							c.Event.Emit(publishEventName, c, rec, 0)
						}
					}()

					for i := totalErrorAggregations - 1; i >= 0; i-- {
						aggregation := errorAggregationOperators[i]
						rec = aggregation(c, rec)
					}
				}

				// Execute exception filters if any
				// normally this one always ok
				// since we always set global exception filter as default
				if _, ok := ws.catchFnsMap[publishEventName]; ok && rec != nil {

					// 3rd param is index of catch function
					c.Event.Emit(publishEventName, c, rec, 0)
				}

				// reset ErrorAggregationOperators
				// to prevent duplicate error aggregation
				// due to error will be added
				// whenever interceptor triggered
				// but WS 1 connection use 1 ctx
				newCtx := context.WithValue(c.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), nil)
				c.Request = c.WithContext(newCtx)

				// clean all events before recursion
				// prevent emit duplicate event
				for _, eventName := range wsSubscribedEvents {
					ws.removeWSEvent(eventName, wsid, c)
				}

				// recursion to keep connection alive
				ws.handleRequest(wsConn, c)
			}
		}()

		c.WS.Message = wsMsg
		publishEventName = common.ToWSEventName(wsInstance.GetSubprotocol(), wsMsg.Event)

		if handlers, isMatched := ws.eventMap[publishEventName]; isMatched {
			for index, handler := range handlers {
				if isNext {
					isNext = false
					handler(c)

					// when ran through all middlewares
					// then invoke mainhandler
					if index == len(handlers)-1 && isNext {
						injectableHandler := ws.mainHandlerMap[publishEventName]

						// data return from main handler
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
								aggregation := aggregations[i]

								if aggregation.IsMainHandlerCalled {

									// set data from main handler into
									// first interceptor
									if i == totalAggregations-1 && len(data) > 1 {
										aggregatedData = data[1].Interface()
									}

									aggregation.SetMainData(aggregatedData)
									aggregatedData = aggregation.Aggregate(c)
								} else {
									isMainHandlerCalled = false
									wsMsg := toWSMessage(reflect.ValueOf(aggregation.InterceptorData))
									ws.publishWSEvent(configPublishedEventName, wsMsg, c)
									break
								}
							}

							if isMainHandlerCalled {
								wsMsg := toWSMessage(reflect.ValueOf(aggregatedData))
								ws.publishWSEvent(configPublishedEventName, wsMsg, c)
							}
						} else {
							if len(data) > 1 {
								wsMsg := toWSMessage(data[1])
								ws.publishWSEvent(configPublishedEventName, wsMsg, c)
							}
						}
					}
				}
			}
		} else {
			ws.invokeGlobalMiddlewares(c, exception.NotFoundException("Cannot emit "+wsMsg.Event+" event"))
		}
	}
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
