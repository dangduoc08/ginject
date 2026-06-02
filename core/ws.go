package core

import (
	"context"
	"encoding/json"
	"io"
	stdHTTP "net/http"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/matcher"
	"golang.org/x/net/websocket"
)

type compiledWS struct {
	pattern matcher.Pattern
}

type WS struct {
	eventToIDMu sync.RWMutex

	eventMap          map[string][]ctx.Handler
	compiledPatterns  []compiledWS
	mainHandlerMap    map[string]any
	eventToID         map[string][]string
	catchFnsMap       map[string][]common.Catch
	globalMiddlewares *[]common.MiddlewareFn

	corsAllowOrigin func(origin string) bool
	invokeHandler   func(f any, c *ctx.Context) []reflect.Value
}

func kindPriority(k matcher.Kind) int {
	switch k {
	case matcher.KindExact:
		return 0
	case matcher.KindSingleSuffix:
		return 1
	case matcher.KindComplex:
		return 2
	default:
		return 3
	}
}

func (ws *WS) buildCompiledPatterns() {
	ws.compiledPatterns = make([]compiledWS, 0, len(ws.eventMap))
	for raw := range ws.eventMap {
		ws.compiledPatterns = append(ws.compiledPatterns, compiledWS{
			pattern: matcher.Parse(raw),
		})
	}
	sort.Slice(ws.compiledPatterns, func(i, j int) bool {
		return kindPriority(ws.compiledPatterns[i].pattern.Kind()) <
			kindPriority(ws.compiledPatterns[j].pattern.Kind())
	})
}

func (ws *WS) matchEventKey(event string) (string, bool) {
	for _, cp := range ws.compiledPatterns {
		if matcher.Match(cp.pattern, event) {
			return cp.pattern.Raw(), true
		}
	}
	return "", false
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

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, c *ctx.Context) {
	s := websocket.Server{
		Handler: websocket.Handler(func(wsConn *websocket.Conn) {
			ws.handleRequest(wsConn, c)
		}),
		Handshake: func(cfg *websocket.Config, _ *stdHTTP.Request) error {
			if ws.corsAllowOrigin == nil || cfg.Origin == nil {
				return nil
			}
			if ws.corsAllowOrigin(cfg.Origin.String()) {
				return nil
			}
			return stdHTTP.ErrAbortHandler
		},
	}
	s.ServeHTTP(w, r)
}

func (ws *WS) handleRequest(wsConn *websocket.Conn, c *ctx.Context) {
	wsInstance := ctx.NewWS(wsConn)
	c.WS = wsInstance
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
		ws.addWSEvent(subscribedEventName, wsid, c, func(s string) {
			_ = wsInstance.SendToConn(c, wsConn, s)
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

		if ws.dispatchMessage(c, wsConn, wsMsg, &isNext, wsid, wsSubscribedEvents) {
			return
		}
	}
}

func (ws *WS) dispatchMessage(c *ctx.Context, wsConn *websocket.Conn, wsMsg ctx.WSMessage, isNext *bool, wsid string, wsSubscribedEvents []string) (recurse bool) {
	incomingEvent := wsMsg.Event
	matchedKey, isMatched := ws.matchEventKey(incomingEvent)

	defer func() {
		if rec := recover(); rec != nil {
			if errorAggregationOperators, ok := c.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)).([]aggregation.AggregationOperator); ok {
				totalErrorAggregations := len(errorAggregationOperators)

				defer func() {
					if rec := recover(); rec != nil {
						_ = c.Broker.Publish(matchedKey, catchEventPayload{reqCtx: c, recovered: rec, index: 0})
					}
				}()

				for i := totalErrorAggregations - 1; i >= 0; i-- {
					op := errorAggregationOperators[i]
					rec = op(c, rec)
				}
			}

			if _, ok := ws.catchFnsMap[matchedKey]; ok && rec != nil {
				_ = c.Broker.Publish(matchedKey, catchEventPayload{reqCtx: c, recovered: rec, index: 0})
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

	if isMatched {
		for index, handler := range ws.eventMap[matchedKey] {
			if *isNext {
				*isNext = false
				handler(c)

				if index == len(ws.eventMap[matchedKey])-1 && *isNext {
					injectableHandler := ws.mainHandlerMap[matchedKey]

					data := ws.invokeHandler(injectableHandler, c)
					if len(data) == 1 {
						data = append(data, reflect.ValueOf("*"))
						data[1], data[0] = data[0], data[1]
					}
					configPublishedEventName := data[0].String()

					if aggregations, ok := c.Context().Value(WithValueKey(matchedKey)).([]*aggregation.Aggregation); ok {
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
		ws.invokeGlobalMiddlewares(c, exception.NotFoundException("Cannot emit "+incomingEvent+" event"))
	}
	return false
}

func (ws *WS) addWSEvent(subscribedEventName, wsid string, c *ctx.Context, cb func(string)) {
	_, _ = c.Broker.Subscribe(subscribedEventName+wsid, func(m *broker.Message) {
		cb(m.Payload.(string))
	})
	ws.eventToIDMu.Lock()
	ws.eventToID[subscribedEventName] = append(ws.eventToID[subscribedEventName], wsid)
	ws.eventToIDMu.Unlock()
}

func (ws *WS) removeWSEvent(subscribedEventName, wsid string, c *ctx.Context) {
	_ = c.Broker.Off(subscribedEventName + wsid)
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
		_ = c.Broker.Publish(configPublishedEventName+wsid, wsMsg)
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
