package core

import (
	"errors"
	stdHTTP "net/http"
	"reflect"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/trace"
	"github.com/dangduoc08/ginject/wsevent"
	"golang.org/x/net/websocket"
)

var errWSHandshakeRejected = errors.New("ws handshake rejected: middleware chain did not call next()")

type WSConfig struct {
	Path              string
	globalMiddlewares []common.MiddlewareFn
	injectedProviders map[string]Provider
	logger            common.Logger
	event             *event.Event

	resolveAndCallHandler func(f any, c *ctx.WSContext) []reflect.Value
	newCtx                func() *ctx.WSContext
	releaseCtx            func(c *ctx.WSContext)
}

type WS struct {
	catchFnsByEvent       map[string][]common.WSCatch
	resolveAndCallHandler func(f any, c *ctx.WSContext) []reflect.Value
	connmgr               *WSConnmgr
	path                  string
	globalMiddlewares     []ctx.HTTPHandler
	injectedProviders     map[string]Provider
	logger                common.Logger
	eventMatcher          *wsevent.WSEvent
	newCtx                func() *ctx.WSContext
	releaseCtx            func(c *ctx.WSContext)
}

func NewWS(cfg *WSConfig) *WS {
	path := str.Enclose("ws", '/')
	if cfg.Path != "" {
		path = str.Enclose(cfg.Path, '/')
	}

	ws := WS{
		catchFnsByEvent:       make(map[string][]common.WSCatch),
		resolveAndCallHandler: cfg.resolveAndCallHandler,
		eventMatcher:          wsevent.NewWSEvent(),
		connmgr:               NewWSConnmgr(cfg.logger),
		path:                  path,
		globalMiddlewares:     resolveGlobalMiddlewares(cfg.globalMiddlewares, cfg.injectedProviders, cfg.event),
		injectedProviders:     cfg.injectedProviders,
		logger:                cfg.logger,
		newCtx:                cfg.newCtx,
		releaseCtx:            cfg.releaseCtx,
	}

	return &ws
}

func resolveGlobalMiddlewares(middlewares []common.MiddlewareFn, injectedProviders map[string]Provider, ev *event.Event) []ctx.HTTPHandler {
	resolved := make([]ctx.HTTPHandler, len(middlewares))
	for i, gm := range middlewares {
		name := reflect.TypeOf(gm).String()
		newGM, err := injectDependencies(gm, "middleware", injectedProviders)
		if err != nil {
			panic(err)
		}
		gm = common.Construct(newGM.Interface(), "NewMiddleware").(common.MiddlewareFn)
		resolved[i] = buildUseMiddleware(gm.Use, ev, name, trace.TransportHTTP)
	}

	return resolved
}

func (ws *WS) isWSPath(p string) bool {
	return str.Enclose(p, '/') == ws.path
}

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, s websocket.Server) {
	s.ServeHTTP(w, r)
}

func (ws *WS) handshake(c *ctx.HTTPContext) error {
	isNext := true
	c.Next = func() {
		isNext = true
	}

	for _, gm := range ws.globalMiddlewares {
		if isNext {
			isNext = false
			gm(c)
		}
	}

	if isNext {
		return nil
	}

	return errWSHandshakeRejected
}

func (ws *WS) handleRequest(wsConn *websocket.Conn) {
	defer func() {
		if err := wsConn.Close(); err != nil {
			ws.logger.Error("WSConnCloseFailed", "error", err)
		}
	}()

	connID, err := crypto.UUID()
	if err != nil {
		ws.logger.Error("WSConnIDGenerationFailed", "error", err)
		return
	}

	if err := websocket.JSON.Send(wsConn, WSPayload{
		ID:   connID,
		Type: TypeConnected,
	}); err != nil {
		ws.logger.Error("WSHandshakeSendFailed", "error", err)
		return
	}

	conn := ws.connmgr.Register(connID, wsConn)
	defer ws.connmgr.Unregister(connID)

	readLoop(conn, ws)
}
