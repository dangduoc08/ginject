package ctx

import (
	"encoding/json"
	"strings"

	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/utils"
	"golang.org/x/net/websocket"
)

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
