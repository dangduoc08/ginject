package core

import (
	"errors"
	stdHTTP "net/http"
	"reflect"
	"time"

	"github.com/dangduoc08/ginject/broker"
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
	broker            *broker.Broker

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
	emitPostInterceptor   func(c *ctx.WSContext, name string, duration time.Duration)
	emitComplete          func(c *ctx.WSContext, operation, target string)
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
		connmgr:               NewWSConnmgr(cfg.logger, cfg.broker),
		path:                  path,
		globalMiddlewares:     resolveGlobalMiddlewares(cfg.globalMiddlewares, cfg.injectedProviders, cfg.event),
		injectedProviders:     cfg.injectedProviders,
		logger:                cfg.logger,
		newCtx:                cfg.newCtx,
		releaseCtx:            cfg.releaseCtx,
	}

	ws.connmgr.startDeadConnDetection(15*time.Second, 60*time.Second)

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
		_ = wsConn.Close()
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

	done := make(chan struct{})
	go pingLoop(wsConn, conn, done, ws.logger)

	readLoop(conn, ws)
	close(done)
}

func pingLoop(wsConn *websocket.Conn, conn *WSConnection, done <-chan struct{}, logger common.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if wsConn == nil {
				return
			}

			if err := wsConn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				logger.Error("WSSetWriteDeadlineFailed", "error", err)
				return
			}

			if err := websocket.JSON.Send(wsConn, WSPayload{Type: TypePing}); err != nil {
				logger.Error("WSPingSendFailed", "error", err)
				return
			}

			if err := wsConn.SetWriteDeadline(time.Time{}); err != nil {
				logger.Error("WSClearWriteDeadlineFailed", "error", err)
				return
			}
		}
	}
}
