package ws

import (
	"fmt"
	"io"
	stdHTTP "net/http"
	"reflect"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/crypto"
	"golang.org/x/net/websocket"
)

type WS struct {
	eventMap          map[string][]ctx.Handler
	mainHandlerMap    map[string]any
	eventToID         map[string][]string
	catchFnsMap       map[string][]common.Catch
	globalMiddlewares *[]common.MiddlewareFn

	corsAllowOrigin func(origin string) bool
	invokeHandler   func(f any, c *ctx.Context) []reflect.Value

	connmgr *WSConnmgr
}

func NewWS() *WS {
	ws := WS{
		catchFnsMap:    make(map[string][]common.Catch),
		eventMap:       make(map[string][]func(*ctx.Context)),
		mainHandlerMap: make(map[string]any),
		eventToID:      make(map[string][]string),
		connmgr:        NewWSConnmgr(),
	}

	return &ws
}

func (ws *WS) Upgrade(w stdHTTP.ResponseWriter, r *stdHTTP.Request, s websocket.Server) {
	s.ServeHTTP(w, r)
}

// TODO: handle handshake auth here
func (ws *WS) Handshake(cfg *websocket.Config, _ *stdHTTP.Request) error {
	if ws.corsAllowOrigin == nil || cfg.Origin == nil {
		return nil
	}
	if ws.corsAllowOrigin(cfg.Origin.String()) {
		return nil
	}
	return stdHTTP.ErrAbortHandler
}

func (ws *WS) HandleRequest(wsConn *websocket.Conn) {
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
			for _, topic := range data.Topic {
				fmt.Println(topic)
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
			}
		}
	}
}
