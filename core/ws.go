package core

import (
	"fmt"
	"io"
	stdHTTP "net/http"
	"reflect"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/internal/str"
	"golang.org/x/net/websocket"
)

type WSConfig struct {
	Path              string
	globalMiddlewares []common.MiddlewareFn
	injectedProviders map[string]Provider
}

type WS struct {
	middlewaresByEvent map[string][]ctx.Handler
	handlerByEvent     map[string]any
	// eventToID         map[string][]string
	catchFnsByEvent map[string][]common.Catch

	resolveAndCallHandler func(f any, c *ctx.Context) []reflect.Value

	// connmgr     *WSConnmgr

	path              string
	globalMiddlewares []common.MiddlewareFn
	injectedProviders map[string]Provider
}

func NewWS(cfg *WSConfig) *WS {
	path := str.Enclose("ws", '/')
	if cfg.Path != "" {
		path = str.Enclose(cfg.Path, '/')
	}

	ws := WS{
		catchFnsByEvent:    make(map[string][]common.Catch),
		middlewaresByEvent: make(map[string][]func(*ctx.Context)),
		handlerByEvent:     make(map[string]any),
		// eventToID:      make(map[string][]string),
		// connmgr:        NewWSConnmgr(),

		path:              path,
		globalMiddlewares: resolveGlobalMiddlewares(cfg.globalMiddlewares, cfg.injectedProviders),
		injectedProviders: cfg.injectedProviders,
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

func (ws *WS) handshake(c *ctx.Context) error {
	isNext := true
	c.Next = func() {
		isNext = true
	}

	for _, gm := range ws.globalMiddlewares {
		if isNext {
			isNext = false
			gm.Use(c, c.Next)
		}
	}

	if isNext {
		if err := c.Broker.Publish(ctx.RequestFinished, c); err != nil {
			fmt.Println(err.Error())
		}
		return nil
	}

	return stdHTTP.ErrAbortHandler
}

func (ws *WS) handleRequest(wsConn *websocket.Conn) {
	defer wsConn.Close()

	id, _ := crypto.UUID()
	err := websocket.JSON.Send(wsConn, WSPayload{
		ID:   id,
		Type: TypeConnected,
	})

	if err != nil {
		fmt.Println(err.Error())
	}

	for {
		go read(wsConn)

		go write(wsConn)

		var data WSPayload

		if err := websocket.JSON.Receive(wsConn, &data); err != nil {
			if err != io.EOF {
				fmt.Println(err.Error())
			}
			return
		}

		if data.Type == TypeSubscribe {
			// for _, topic := range data.Topic {
			// 	fmt.Println(topic)
			// GlobalBroker.Subscribe(topic, func(m *broker2.Message) {
			// 	fmt.Println("vao day")

			// 	err := websocket.JSON.Send(wsConn, WSPayload{
			// 		ID:      id,
			// 		Type:    TypePublish,
			// 		Topic:   []string{m.Topic},
			// 		Message: m.Payload,
			// 	})

			// 	if err != nil {
			// 		panic(err)
			// 	}
			// })
			// }
		}
	}
}
