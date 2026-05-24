package ctx

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/utils"
	"golang.org/x/net/websocket"
)

// WSContext carries WebSocket connection state for a single connection lifetime.
// Its ExecutionContext is rooted at context.Background() — NOT the HTTP upgrade
// request context, which is cancelled when the upgrade handshake completes.
//
// Call Cancel() or defer the cancel func returned by NewWSContext when the
// connection closes; this propagates cancellation to all per-message goroutines.
type WSContext struct {
	exec   *ExecutionContext
	cancel context.CancelFunc

	Conn    *websocket.Conn
	ConnID  string
	Message WSMessage
}

// NewWSContext creates a WSContext with its own independent ExecutionContext.
// The caller must defer the returned cancel func (or call ws.Cancel()) on
// connection close to propagate cancellation to all derived contexts.
func NewWSContext(conn *websocket.Conn) (*WSContext, context.CancelFunc) {
	exec, cancel := NewWSExecutionContext()

	uuid, _ := utils.StrUUID()
	key := conn.Request().Header.Get("Sec-Websocket-Key")
	connID := key + strings.ReplaceAll(uuid, "-", "")

	ws := &WSContext{
		exec:   exec,
		cancel: cancel,
		Conn:   conn,
		ConnID: connID,
	}
	exec.Set(MetaRequestID, connID)

	return ws, cancel
}

// Exec returns the connection-scoped ExecutionContext.
// Use it to check cancellation or store connection-scoped metadata.
func (ws *WSContext) Exec() *ExecutionContext { return ws.exec }

// Cancel signals all derived per-message contexts and any GoSafe goroutines
// spawned from this connection to stop. Called automatically by the framework
// on disconnect via the deferred cancel returned by NewWSContext.
func (ws *WSContext) Cancel() { ws.cancel() }

// NewMessageContext derives a short-lived child ExecutionContext for
// processing a single incoming message with a per-message SLA timeout.
// The child is cancelled when d elapses or the connection-level exec
// is cancelled, whichever comes first. Caller must defer cancel().
func (ws *WSContext) NewMessageContext(d time.Duration) (*ExecutionContext, context.CancelFunc) {
	return WithTimeout(ws.exec, d)
}

type WSMessage struct {
	Event   string    `json:"event"`
	Payload WSPayload `json:"payload"`
}

type WS struct {
	uuid       string
	Connection *websocket.Conn
	Message    WSMessage
}

func NewWS(wsConn *websocket.Conn) *WS {
	ws := &WS{
		Connection: wsConn,
	}

	if ws.uuid == "" {
		uuid, err := utils.StrUUID()
		if err != nil {
			panic(err)
		}
		ws.uuid = uuid
	}

	return ws
}

func (ws *WS) GetSubprotocol() string {
	proto := ws.Connection.Config().Protocol
	if len(proto) == 0 {
		return "*"
	}
	return proto[0]
}

func (ws *WS) GetSubscribedEvents() []string {
	wsSubscribedEvents := strings.Split(ws.Connection.Request().URL.Query().Get("events"), ",")
	wsSubscribedEvents = append(wsSubscribedEvents, "*")
	wsSubscribedEvents = utils.ArrFilter(wsSubscribedEvents, func(el string, i int) bool {
		return strings.TrimSpace(el) != ""
	})
	wsSubscribedEvents = utils.ArrToUnique(wsSubscribedEvents)
	return wsSubscribedEvents
}

func (ws *WS) GetConnID() string {
	wsID := ws.Connection.Request().Header.Get("Sec-Websocket-Key")
	return wsID + strings.ReplaceAll(ws.uuid, "-", "")
}

func (ws *WS) CanEstablish(insertedEvents map[string]string) bool {
	requestSubprotocol := ws.GetSubprotocol()
	for eventname := range insertedEvents {
		configSubprotocol, _ := ResolveWSEventname(eventname)
		if requestSubprotocol == configSubprotocol {
			return true
		}
	}

	return false
}

// Use for return error response to
// itself connection
func (ws *WS) SendSelf(c *Context, message any) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		panic(exception.InternalServerErrorException(err.Error()))
	}

	err = websocket.Message.Send(ws.Connection, string(jsonData))
	if err != nil {
		panic(exception.InternalServerErrorException(err.Error()))
	}

	c.Event.Emit(REQUEST_FINISHED, c)
	return nil
}

func (ws *WS) SendToConn(c *Context, wsConn *websocket.Conn, message string) error {
	err := websocket.Message.Send(wsConn, message)
	if err != nil {
		panic(exception.InternalServerErrorException(err.Error()))
	}

	c.Event.Emit(REQUEST_FINISHED, c)
	return nil
}
