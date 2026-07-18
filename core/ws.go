package core

import (
	"errors"
	stdHTTP "net/http"
	"reflect"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/wsevent"
	"golang.org/x/net/websocket"
)

var errWSHandshakeRejected = errors.New("ws handshake rejected: middleware chain did not call next()")

type WSConfig struct {
	Path              string
	globalMiddlewares []common.MiddlewareFn
	injectedProviders map[string]Provider
	logger            common.Logger

	resolveAndCallHandler func(f any, c *ctx.WSContext) []reflect.Value
	newCtx                func() *ctx.WSContext
	releaseCtx            func(c *ctx.WSContext)
}

type WS struct {
	catchFnsByEvent       map[string][]common.Catch
	resolveAndCallHandler func(f any, c *ctx.WSContext) []reflect.Value
	connmgr               *WSConnmgr
	path                  string
	globalMiddlewares     []common.MiddlewareFn
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
		catchFnsByEvent:       make(map[string][]common.Catch),
		resolveAndCallHandler: cfg.resolveAndCallHandler,
		eventMatcher:          wsevent.NewWSEvent(),
		connmgr:               NewWSConnmgr(cfg.logger),
		path:                  path,
		globalMiddlewares:     resolveGlobalMiddlewares(cfg.globalMiddlewares, cfg.injectedProviders),
		injectedProviders:     cfg.injectedProviders,
		logger:                cfg.logger,
		newCtx:                cfg.newCtx,
		releaseCtx:            cfg.releaseCtx,
	}

	return &ws
}

func resolveGlobalMiddlewares(middlewares []common.MiddlewareFn, injectedProviders map[string]Provider) []common.MiddlewareFn {
	resolved := make([]common.MiddlewareFn, len(middlewares))
	for i, gm := range middlewares {
		newGM, err := injectDependencies(gm, "middleware", injectedProviders)
		if err != nil {
			panic(err)
		}
		resolved[i] = common.Construct(newGM.Interface(), "NewMiddleware").(common.MiddlewareFn)
	}

	return resolved
}

func (ws *WS) isWSPath(p string) bool {
	return str.Enclose(p, '/') == ws.path
}

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, s websocket.Server) {
	s.ServeHTTP(w, r)
}

func (ws *WS) handshake(c *ctx.WSContext) error {
	isNext := true
	c.Next = func() {
		isNext = true
	}

	for _, gm := range ws.globalMiddlewares {
		if isNext {
			isNext = false
			gm.Use(c.Request, c.ResponseWriter, c.Next)
		}
	}

	if isNext {
		if err := c.Broker.Publish(ctx.RequestFinished, c); err != nil {
			ws.logger.Error("WSRequestFinishedPublishFailed", "error", err)
		}
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
