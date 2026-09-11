package core

import (
	"errors"
	stdHTTP "net/http"
	"reflect"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/internal/crypto"
	"strings"

	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/memorybroker"
	"github.com/dangduoc08/ginject/trace"
	"github.com/dangduoc08/ginject/wsevent"
	"golang.org/x/net/websocket"
)

var errWSHandshakeRejected = errors.New("ws handshake rejected: middleware chain did not call next()")
var errWSOriginRejected = errors.New("ws handshake rejected: origin not in WSConfig.AllowedOrigins")

type WSConfig struct {
	Path              string
	MaxConnections    int
	MaxPayloadBytes   int
	WriteTimeout      time.Duration
	AllowedOrigins    []string
	globalMiddlewares []common.MiddlewareFn
	injectedProviders map[string]Provider
	logger            common.Logger
	event             *event.Event
	broker            *memorybroker.Broker
	shutdownChan      <-chan struct{}

	resolveAndCallHandler func(f any, c *ctx.WSContext) []reflect.Value
	newCtx                func() *ctx.WSContext
	releaseCtx            func(c *ctx.WSContext)
}

type WS struct {
	maxPayloadBytes       int
	allowedOrigins        map[string]bool
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

	maxPayloadBytes := DefaultWSMaxPayloadBytes
	if cfg.MaxPayloadBytes > 0 {
		maxPayloadBytes = cfg.MaxPayloadBytes
	}

	var allowedOrigins map[string]bool
	if len(cfg.AllowedOrigins) > 0 {
		allowedOrigins = make(map[string]bool, len(cfg.AllowedOrigins))
		for _, origin := range cfg.AllowedOrigins {
			allowedOrigins[strings.TrimSuffix(origin, "/")] = true
		}
	}

	ws := WS{
		maxPayloadBytes:       maxPayloadBytes,
		allowedOrigins:        allowedOrigins,
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

	if cfg.MaxConnections > 0 {
		ws.connmgr.maxConns = cfg.MaxConnections
	}
	if cfg.WriteTimeout > 0 {
		ws.connmgr.writeTimeout = cfg.WriteTimeout
	}

	ws.connmgr.startDeadConnDetection(15*time.Second, 60*time.Second, cfg.shutdownChan)

	if allowedOrigins == nil && len(cfg.globalMiddlewares) == 0 && cfg.logger != nil {
		cfg.logger.Warn("WSOriginUnrestricted",
			"reason", "no WSConfig.AllowedOrigins and no middleware to check Origin; every site can open a WebSocket with the visitor's cookies",
			"fix", "set WSConfig.AllowedOrigins, or pass a configured cors.CORS to EnableWS")
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

// isOriginAllowed enforces WSConfig.AllowedOrigins when configured. An absent
// Origin is allowed: it marks a non-browser client, which the header cannot
// protect against anyway.
func (ws *WS) isOriginAllowed(r *stdHTTP.Request) bool {
	if ws.allowedOrigins == nil {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	return ws.allowedOrigins[strings.TrimSuffix(origin, "/")]
}

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, s websocket.Server) {
	s.ServeHTTP(w, r)
}

func (ws *WS) handshake(c *ctx.HTTPContext) error {
	if !ws.isOriginAllowed(c.Request) {
		return errWSOriginRejected
	}

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

	wsConn.MaxPayloadBytes = ws.maxPayloadBytes

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
	if conn == nil {
		ws.logger.Warn("WSConnectionRejected", "id", connID, "connections", ws.connmgr.Count())
		return
	}
	defer ws.connmgr.Unregister(connID)

	done := make(chan struct{})
	go pingLoop(conn, done)

	readLoop(conn, ws)
	close(done)
}

func pingLoop(conn *WSConnection, done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			conn.TrySend(WSPayload{Type: TypePing})
		}
	}
}
