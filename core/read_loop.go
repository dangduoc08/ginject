package core

import (
	"io"
	"time"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/memorybroker"
	"github.com/dangduoc08/ginject/trace"
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
	TypeResponse    WSPayloadType = "response"
	TypeAck         WSPayloadType = "ack"
	TypeError       WSPayloadType = "error"
	TypePing        WSPayloadType = "ping"
	TypePong        WSPayloadType = "pong"
)

const maxWSTopicLength = 255

type WSPayload struct {
	Message any           `json:"message"`
	Type    WSPayloadType `json:"type"`
	ID      string        `json:"id"`
	Topic   []string      `json:"topic,omitempty"`
	Pattern string        `json:"pattern,omitempty"`
}

type WSTopicResult struct {
	Topic    string `json:"topic"`
	Status   string `json:"status"`
	Code     int    `json:"code,omitempty"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
	reported bool
}

func readLoop(conn *WSConnection, ws *WS) {
	for {
		var payload WSPayload
		if err := websocket.JSON.Receive(conn.Conn, &payload); err != nil {
			if err != io.EOF {
				ws.logger.Error("WSReadFailed", "id", conn.ID, "error", err)
			}
			return
		}

		ws.connmgr.touch(conn.ID)

		switch payload.Type {
		case TypeSubscribe:
			handleSubscribe(conn, ws, payload)
		case TypeUnsubscribe:
			handleUnsubscribe(conn, ws, payload)
		case TypePublish:
			handlePublish(conn, ws, payload)
		case TypePing:
			ws.connmgr.trySend(conn, WSPayload{ID: payload.ID, Type: TypePong})
		case TypePong:
		default:
			replyException(conn, ws, payload.ID, exception.ProtocolErrorException("unsupported type: "+string(payload.Type)))
		}
	}
}

func validateTopic(topic string) *exception.Exception {
	if topic == "" {
		ex := exception.InvalidPayloadException("topic must not be empty")
		return &ex
	}
	if len(topic) > maxWSTopicLength {
		ex := exception.MessageTooBigException("topic exceeds the maximum length of 255 characters")
		return &ex
	}
	return nil
}

func newOpCtx(conn *WSConnection, ws *WS, pattern, topic, operation string, payload WSPayload) *ctx.WSContext {
	c := ws.newCtx()
	c.Init(conn.Conn)
	c.SetEvent(conn.ID, pattern, topic, operation)
	if req := conn.Conn.Request(); req != nil {
		c.SetContext(req.Context())
	}
	messageMap, _ := payload.Message.(map[string]any)
	c.SetWSPayload(ctx.WSPayload(messageMap))

	return c
}

func failure(ex exception.Exception) WSTopicResult {
	return WSTopicResult{
		Status:  trace.StatusRejected,
		Code:    ex.GetCode(),
		Error:   ex.Error(),
		Message: ex.GetMessage(),
	}
}

func handleSubscribe(conn *WSConnection, ws *WS, payload WSPayload) {
	results := make([]WSTopicResult, 0, len(payload.Topic))

	for _, topic := range payload.Topic {
		result := WSTopicResult{Topic: topic, Status: trace.StatusOK}

		if ex := validateTopic(topic); ex != nil {
			result = failure(*ex)
			result.Topic = topic
			results = append(results, result)
			emitOpComplete(conn, ws, "", topic, trace.OperationSubscribe, trace.StatusRejected, ex.GetCode(), payload)
			continue
		}

		if ws.connmgr.isSubscribed(conn.ID, topic) {
			results = append(results, result)
			continue
		}

		item, pattern, ok := ws.eventMatcher.Match(topic)
		if !ok {
			ex := exception.TopicNotFoundException("no handler registered for topic: " + topic)
			result = failure(ex)
			result.Topic = topic
			results = append(results, result)
			emitOpComplete(conn, ws, "", topic, trace.OperationSubscribe, trace.StatusRejected, ex.GetCode(), payload)
			continue
		}

		c, guardOK := runWSGuards(conn, ws, pattern, topic, trace.OperationSubscribe, item, payload)
		if !guardOK {
			ws.emitComplete(c, trace.OperationSubscribe, topic, trace.StatusRejected, exception.ForbiddenException("").GetCode())
			ws.releaseCtx(c)
			result = failure(exception.PolicyViolationException("subscribe denied by guard for topic: " + topic))
			result.Topic = topic
			result.reported = true
			results = append(results, result)
			continue
		}

		if conn.Closed() {
			ws.releaseCtx(c)
			return
		}

		err := ws.connmgr.Subscribe(conn.ID, topic, func(m *memorybroker.Message) {
			ws.connmgr.trySend(conn, WSPayload{
				Type:    TypeEvent,
				Topic:   []string{m.Topic},
				Pattern: topic,
				Message: m.Payload,
			})
		})

		if err != nil {
			var ex exception.Exception
			if err == ErrWSSubscriptionLimit {
				ex = exception.SubscriptionLimitException("subscription limit reached for this connection")
			} else {
				ex = exception.WSInternalErrorException(err.Error())
			}
			result = failure(ex)
			result.Topic = topic
			results = append(results, result)
			ws.emitComplete(c, trace.OperationSubscribe, topic, trace.StatusFailed, ex.GetCode())
			ws.releaseCtx(c)
			continue
		}

		if conn.Closed() {
			_ = ws.connmgr.Unsubscribe(conn.ID, topic)
			ws.releaseCtx(c)
			return
		}

		results = append(results, result)
		ws.emitComplete(c, trace.OperationSubscribe, topic, trace.StatusOK, 0)
		ws.releaseCtx(c)
	}

	replyResults(conn, ws, payload.ID, TypeSubscribe, results)
}

func handleUnsubscribe(conn *WSConnection, ws *WS, payload WSPayload) {
	results := make([]WSTopicResult, 0, len(payload.Topic))

	for _, topic := range payload.Topic {
		result := WSTopicResult{Topic: topic, Status: trace.StatusOK}

		if ex := validateTopic(topic); ex != nil {
			result = failure(*ex)
			result.Topic = topic
			results = append(results, result)
			emitOpComplete(conn, ws, "", topic, trace.OperationUnsubscribe, trace.StatusRejected, ex.GetCode(), payload)
			continue
		}

		if !ws.connmgr.isSubscribed(conn.ID, topic) {
			results = append(results, result)
			emitOpComplete(conn, ws, "", topic, trace.OperationUnsubscribe, trace.StatusOK, 0, payload)
			continue
		}

		item, pattern, matched := ws.eventMatcher.Match(topic)
		var c *ctx.WSContext
		if matched {
			var guardOK bool
			c, guardOK = runWSGuards(conn, ws, pattern, topic, trace.OperationUnsubscribe, item, payload)
			if !guardOK {
				ws.emitComplete(c, trace.OperationUnsubscribe, topic, trace.StatusRejected, exception.ForbiddenException("").GetCode())
				ws.releaseCtx(c)
				result = failure(exception.PolicyViolationException("unsubscribe denied by guard for topic: " + topic))
				result.Topic = topic
				results = append(results, result)
				continue
			}
		} else {
			c = newOpCtx(conn, ws, "", topic, trace.OperationUnsubscribe, payload)
		}

		if err := ws.connmgr.Unsubscribe(conn.ID, topic); err != nil {
			ex := exception.WSInternalErrorException(err.Error())
			result = failure(ex)
			result.Topic = topic
			results = append(results, result)
			ws.emitComplete(c, trace.OperationUnsubscribe, topic, trace.StatusFailed, ex.GetCode())
			ws.releaseCtx(c)
			continue
		}

		results = append(results, result)
		ws.emitComplete(c, trace.OperationUnsubscribe, topic, trace.StatusOK, 0)
		ws.releaseCtx(c)
	}

	replyResults(conn, ws, payload.ID, TypeUnsubscribe, results)
}

func handlePublish(conn *WSConnection, ws *WS, payload WSPayload) {
	results := make([]WSTopicResult, 0, len(payload.Topic))

	for _, topic := range payload.Topic {
		result := WSTopicResult{Topic: topic, Status: trace.StatusOK}

		if ex := validateTopic(topic); ex != nil {
			result = failure(*ex)
			result.Topic = topic
			results = append(results, result)
			emitOpComplete(conn, ws, "", topic, trace.OperationPublish, trace.StatusRejected, ex.GetCode(), payload)
			continue
		}

		item, pattern, ok := ws.eventMatcher.Match(topic)
		if !ok {
			ex := exception.TopicNotFoundException("no handler registered for topic: " + topic)
			result = failure(ex)
			result.Topic = topic
			results = append(results, result)
			emitOpComplete(conn, ws, "", topic, trace.OperationPublish, trace.StatusRejected, ex.GetCode(), payload)
			continue
		}

		if !ws.connmgr.isSubscribed(conn.ID, topic) {
			ex := exception.NotSubscribedException("must subscribe before publishing to: " + topic)
			result = failure(ex)
			result.Topic = topic
			results = append(results, result)
			emitOpComplete(conn, ws, pattern, topic, trace.OperationPublish, trace.StatusRejected, ex.GetCode(), payload)
			continue
		}

		if !dispatchWSEvent(conn, ws, pattern, topic, item, payload) {
			result = failure(exception.PolicyViolationException("publish rejected for topic: " + topic))
			result.Topic = topic
			result.reported = true
			results = append(results, result)
			continue
		}

		if err := (*ws.connmgr.Broker).Publish(topic, payload.Message); err != nil {
			ex := exception.WSInternalErrorException(err.Error())
			result = failure(ex)
			result.Topic = topic
			results = append(results, result)
			ws.logger.Error("WSBrokerPublishFailed", "id", conn.ID, "topic", topic, "error", err)
			continue
		}

		results = append(results, result)
	}

	replyResults(conn, ws, payload.ID, TypePublish, results)
}

func replyResults(conn *WSConnection, ws *WS, id string, t WSPayloadType, results []WSTopicResult) {
	hasFailure := false
	allReported := len(results) > 0
	topics := make([]string, 0, len(results))
	for _, r := range results {
		topics = append(topics, r.Topic)
		if r.Status != trace.StatusOK {
			hasFailure = true
		}
		if !r.reported {
			allReported = false
		}
	}

	if allReported {
		return
	}

	replyType := TypeAck
	if hasFailure {
		replyType = TypeError
	}

	ws.connmgr.trySend(conn, WSPayload{
		ID:      id,
		Type:    replyType,
		Topic:   topics,
		Message: results,
	})
}

func emitOpComplete(conn *WSConnection, ws *WS, pattern, topic, operation, status string, code int, payload WSPayload) {
	c := newOpCtx(conn, ws, pattern, topic, operation, payload)
	ws.emitComplete(c, operation, topic, status, code)
	ws.releaseCtx(c)
}

func runCatchChain(conn *WSConnection, ws *WS, c *ctx.WSContext, pattern string, payloadID string, rec any) {
	c.SetSend(func(data any) {
		reply(conn, ws, TypeError, payloadID, c.Topic(), data)
	})

	if catchFns, ok := ws.catchFnsByEvent[pattern]; ok {
		common.RunWSCatchChain(c, catchFns, rec)
		return
	}

	replyException(conn, ws, payloadID, *common.NormalizeRecovered(rec))
}

func replyException(conn *WSConnection, ws *WS, id string, ex exception.Exception) {
	reply(conn, ws, TypeError, id, "", ctx.Map{
		"code":    ex.GetCode(),
		"error":   ex.Error(),
		"message": ex.GetMessage(),
	})
}

func runWSGuards(conn *WSConnection, ws *WS, pattern, topic, operation string, item wsevent.WSEventItem, payload WSPayload) (c *ctx.WSContext, ok bool) {
	c = newOpCtx(conn, ws, pattern, topic, operation, payload)

	defer func() {
		if rec := recover(); rec != nil {
			runCatchChain(conn, ws, c, pattern, payload.ID, rec)
			ok = false
		}
	}()

	isNext := true
	c.Next = func() { isNext = true }

	for _, guard := range item.Middlewares {
		if isNext {
			isNext = false
			guard(c)
		}
	}

	return c, isNext
}

func runWSInterceptors(conn *WSConnection, ws *WS, c *ctx.WSContext, pattern string, item wsevent.WSEventItem, payload WSPayload) (ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			runCatchChain(conn, ws, c, pattern, payload.ID, rec)
			ok = false
		}
	}()

	isNext := true
	c.Next = func() { isNext = true }

	for _, interceptor := range item.Interceptors {
		if isNext {
			isNext = false
			interceptor(c)
		}
	}

	return isNext
}

func dispatchWSEvent(conn *WSConnection, ws *WS, pattern, topic string, item wsevent.WSEventItem, payload WSPayload) (ok bool) {
	c, guardOK := runWSGuards(conn, ws, pattern, topic, trace.OperationPublish, item, payload)
	defer ws.releaseCtx(c)

	status := trace.StatusOK
	code := 0
	defer func() {
		ws.emitComplete(c, trace.OperationPublish, topic, status, code)
	}()

	if !guardOK {
		status = trace.StatusRejected
		code = exception.ForbiddenException("").GetCode()
		return false
	}

	if !runWSInterceptors(conn, ws, c, pattern, item, payload) {
		status = trace.StatusFailed
		code = exception.WSInternalErrorException("").GetCode()
		return false
	}

	defer func() {
		if rec := recover(); rec != nil {
			status = trace.StatusFailed
			code = common.NormalizeRecovered(rec).GetCode()
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
				for j := i; j >= 0; j-- {
					ws.emitPostInterceptor(c, aggregations[j].Name, 0)
				}
				reply(conn, ws, TypeResponse, payload.ID, topic, agg.InterceptorData)
				return true
			}

			if i == totalAggregations-1 && len(data) > 0 {
				aggregatedData = data[len(data)-1].Interface()
			}
			agg.SetMainData(aggregatedData)
			start := time.Now()
			aggregatedData = agg.Aggregate()
			ws.emitPostInterceptor(c, agg.Name, time.Since(start))
		}

		reply(conn, ws, TypeResponse, payload.ID, topic, aggregatedData)
		return true
	}

	if len(data) > 0 {
		reply(conn, ws, TypeResponse, payload.ID, topic, data[len(data)-1].Interface())
	}

	return true
}

func reply(conn *WSConnection, ws *WS, t WSPayloadType, id, topic string, message any) {
	p := WSPayload{ID: id, Type: t, Message: message}
	if topic != "" {
		p.Topic = []string{topic}
	}
	ws.connmgr.trySend(conn, p)
}
