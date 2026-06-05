package core

import (
	"fmt"
	stdHTTP "net/http"
	"reflect"
	"sort"
	"sync"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
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
	fmt.Println(wsConn)
}
