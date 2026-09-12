package core

import (
	"errors"
	stdHTTP "net/http"
	"reflect"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/exception"
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

const (
	DefaultWSPingInterval = 30 * time.Second
	DefaultWSPongTimeout  = 75 * time.Second
	DefaultWSReapInterval = 15 * time.Second
)

type WSConfig struct {
	Path                    string
	MaxConnections          int
	MaxPayloadBytes         int
	MaxSubscriptionsPerConn int
	MaxDroppedFrames        int64
	SendBufferSize          int
	WriteTimeout            time.Duration
	PingInterval            time.Duration
	PongTimeout             time.Duration
	ReapInterval            time.Duration
	AllowedOrigins          []string
	Broker                  memorybroker.Broker

	globalMiddlewares []common.MiddlewareFn
	inheritedHandlers []ctx.HTTPHandler
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
	pingInterval          time.Duration
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
	emitHandler           func(c *ctx.WSContext, name string, duration time.Duration)
	emitComplete          func(c *ctx.WSContext, operation, target, status string, code int)
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

	pingInterval := DefaultWSPingInterval
	if cfg.PingInterval > 0 {
		pingInterval = cfg.PingInterval
	}
	pongTimeout := DefaultWSPongTimeout
	if cfg.PongTimeout > 0 {
		pongTimeout = cfg.PongTimeout
	}
	reapInterval := DefaultWSReapInterval
	if cfg.ReapInterval > 0 {
		reapInterval = cfg.ReapInterval
	}

	var allowedOrigins map[string]bool
	if len(cfg.AllowedOrigins) > 0 {
		allowedOrigins = make(map[string]bool, len(cfg.AllowedOrigins))
		for _, origin := range cfg.AllowedOrigins {
			allowedOrigins[strings.TrimSuffix(origin, "/")] = true
		}
	}

	broker := cfg.broker
	if cfg.Broker != nil {
		b := cfg.Broker
		broker = &b
	}

	globalMiddlewares := append(
		[]ctx.HTTPHandler{},
		cfg.inheritedHandlers...,
	)
	globalMiddlewares = append(
		globalMiddlewares,
		resolveGlobalMiddlewares(cfg.globalMiddlewares, cfg.injectedProviders, cfg.event)...,
	)

	ws := WS{
		maxPayloadBytes:       maxPayloadBytes,
		pingInterval:          pingInterval,
		allowedOrigins:        allowedOrigins,
		catchFnsByEvent:       make(map[string][]common.WSCatch),
		resolveAndCallHandler: cfg.resolveAndCallHandler,
		eventMatcher:          wsevent.NewWSEvent(),
		connmgr:               NewWSConnmgr(cfg.logger, broker),
		path:                  path,
		globalMiddlewares:     globalMiddlewares,
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
	if cfg.SendBufferSize > 0 {
		ws.connmgr.sendBufferSize = cfg.SendBufferSize
	}
	if cfg.MaxSubscriptionsPerConn != 0 {
		ws.connmgr.maxSubsPerConn = cfg.MaxSubscriptionsPerConn
	}
	if cfg.MaxDroppedFrames != 0 {
		ws.connmgr.maxDroppedFrame = cfg.MaxDroppedFrames
	}

	ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {}
	ws.emitHandler = func(c *ctx.WSContext, name string, duration time.Duration) {}
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {}

	ws.connmgr.startDeadConnDetection(reapInterval, pongTimeout, cfg.shutdownChan)

	if allowedOrigins == nil && len(globalMiddlewares) == 0 && cfg.logger != nil {
		cfg.logger.Warn("WSOriginUnrestricted",
			"reason", "no WSConfig.AllowedOrigins and no middleware to check Origin; every site can open a WebSocket with the visitor's cookies",
			"fix", "set WSConfig.AllowedOrigins, or pass a configured cors.CORS to EnableWS")
	}

	return &ws
}

func (ws *WS) Stats() WSStats {
	return ws.connmgr.Stats()
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
		c.Status(stdHTTP.StatusForbidden)
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
		c.Status(stdHTTP.StatusSwitchingProtocols)
		return nil
	}

	if c.Code == stdHTTP.StatusSwitchingProtocols || c.Code == stdHTTP.StatusOK {
		c.Status(stdHTTP.StatusForbidden)
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

	conn := ws.connmgr.Register(connID, wsConn)
	if conn == nil {
		ws.logger.Warn("WSConnectionRejected", "id", connID, "connections", ws.connmgr.Count())
		ex := exception.TryAgainLaterException("connection limit reached")
		_ = websocket.JSON.Send(wsConn, WSPayload{
			Type: TypeError,
			Message: ctx.Map{
				"code":    ex.GetCode(),
				"error":   ex.Error(),
				"message": ex.GetMessage(),
			},
		})
		return
	}
	defer ws.connmgr.Unregister(connID)

	if !conn.TrySend(WSPayload{ID: connID, Type: TypeConnected}) {
		ws.logger.Error("WSHandshakeSendFailed", "id", connID)
		return
	}

	done := make(chan struct{})
	var pingWG sync.WaitGroup
	pingWG.Add(1)
	go func() {
		defer pingWG.Done()
		pingLoop(conn, ws.connmgr, done, ws.pingInterval)
	}()

	readLoop(conn, ws)
	close(done)
	pingWG.Wait()
}

func pingLoop(conn *WSConnection, connmgr *WSConnmgr, done <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			connmgr.trySend(conn, WSPayload{Type: TypePing})
		}
	}
}
