package core

import (
	"fmt"
	"io"
	stdHTTP "net/http"

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
	// eventMap          map[string][]ctx.Handler
	// mainHandlerMap    map[string]any
	// eventToID         map[string][]string
	catchFnsByEvent map[string][]common.Catch

	// invokeHandler   func(f any, c *ctx.Context) []reflect.Value

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
		// catchFnsByEvent:    make(map[string][]common.Catch),
		// eventMap:       make(map[string][]func(*ctx.Context)),
		// mainHandlerMap: make(map[string]any),
		// eventToID:      make(map[string][]string),
		// connmgr:        NewWSConnmgr(),

		path:              path,
		globalMiddlewares: cfg.globalMiddlewares,
	}

	return &ws
}

func (ws *WS) IsWSPath(p string) bool {
	return str.Enclose(p, '/') == ws.path
}

func (ws *WS) upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, s websocket.Server) {
	s.ServeHTTP(w, r)
}

func (ws *WS) handshake(c *ctx.Context) error {
	if len(ws.globalMiddlewares) > 0 {
		isNext := true
		for _, gm := range ws.globalMiddlewares {
			newGM, err := injectDependencies(gm, "middleware", ws.injectedProviders)
			if err != nil {
				panic(err)
			}
			gm = common.Construct(newGM.Interface(), "NewMiddleware").(common.MiddlewareFn)

			c.Next = func() {
				isNext = true
			}
			if isNext {
				isNext = false
				gm.Use(c, c.Next)
			}
		}
		if isNext {
			c.Broker.Publish(ctx.RequestFinished, c)
			return nil
		}
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
