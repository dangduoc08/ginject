package core

import (
	"fmt"
	stdHTTP "net/http"
	"reflect"
	"sort"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/connmgr"
	"github.com/dangduoc08/ginject/ctx"
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

	connectionManager *connmgr.ConnectionManager
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

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, deferFunc func()) {
	s := websocket.Server{
		Handler: websocket.Handler(ws.handleRequest),
		Handshake: func(cfg *websocket.Config, r *stdHTTP.Request) error {
			defer deferFunc()

			return ws.handshake(cfg, r)
		},
	}
	s.ServeHTTP(w, r)
}

func (ws *WS) handshake(cfg *websocket.Config, _ *stdHTTP.Request) error {
	fmt.Println("handshakre")
	if ws.corsAllowOrigin == nil || cfg.Origin == nil {
		return nil
	}
	if ws.corsAllowOrigin(cfg.Origin.String()) {
		return nil
	}
	return stdHTTP.ErrAbortHandler
}

func (ws *WS) handleRequest(wsConn *websocket.Conn) {
	fmt.Println("Zo")
	var msg map[string]any

	websocket.JSON.Receive(wsConn, &msg)

	fmt.Println("ms", msg)

	defer func() {
		wsConn.Close()
	}()

	// for {
	// 	var msg map[string]any

	// 	err := websocket.JSON.Receive(wsConn, &msg)

	// 	fmt.Println(msg)

	// 	err = websocket.Message.Send(wsConn, "hello client")
	// 	if err != nil {
	// 		return
	// 	}
	// }
}
