package core

import (
	"io"
	stdHTTP "net/http"
	"reflect"
	"sort"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/matcher"
	"golang.org/x/net/websocket"
)

// TODO: we need enhancing handshake auth mechanism

// TODO: we will support codec base on sub-protcol passed from client
// but now support json first
/**
type CodecType string

const (
	CodecJSON    CodecType = "json"
	CodecMsgPack CodecType = "msgpack"
	CodecProto   CodecType = "protobuf"
)

**/

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

type compiledWS struct {
	pattern matcher.Pattern
}

type WS struct {
	eventMap          map[string][]ctx.Handler
	compiledPatterns  []compiledWS
	mainHandlerMap    map[string]any
	eventToID         map[string][]string
	catchFnsMap       map[string][]common.Catch
	globalMiddlewares *[]common.MiddlewareFn

	corsAllowOrigin func(origin string) bool
	invokeHandler   func(f any, c *ctx.Context) []reflect.Value

	connmgr *WSConnmgr
}

type WSPayload struct {
	Type  WSPayloadType `json:"type"`
	ID    string        `json:"id"`
	Topic []string      `json:"topic,omitempty"`
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

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, s websocket.Server) {
	s.ServeHTTP(w, r)
}

// TODO: handle handshake auth here
func (ws *WS) handshake(cfg *websocket.Config, _ *stdHTTP.Request) error {
	if ws.corsAllowOrigin == nil || cfg.Origin == nil {
		return nil
	}
	if ws.corsAllowOrigin(cfg.Origin.String()) {
		return nil
	}
	return stdHTTP.ErrAbortHandler
}

// TODO: Layer 2, after handshakre, subscribe topics
func (ws *WS) handleRequest(wsConn *websocket.Conn) {
	logger := log.NewLog(nil)

	defer func() {
		err := wsConn.Close()
		logger.Error(err.Error())

	}()

	// codec := "json"
	// if protocols := wsConn.Config().Protocol; len(protocols) > 0 {
	// 	codec = protocols[0]
	// }

	id, _ := crypto.UUID()
	err := websocket.JSON.Send(wsConn, WSPayload{
		ID:   id,
		Type: TypeConnected,
	})

	if err != nil {
		panic(err)
	}

	for {
		var data WSPayload

		if err := websocket.JSON.Receive(wsConn, &data); err != nil {
			if err != io.EOF {
				logger.Error(err.Error())
			}
			return
		}

		logger.Info("Message", "type", data.Type, "id", data.ID, "topic", data.Topic)
	}
}
