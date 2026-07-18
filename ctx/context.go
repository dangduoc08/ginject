package ctx

import (
	"net/http"
	"time"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/internal/crypto"
	"golang.org/x/net/websocket"
)

type (
	Map         map[string]any
	ErrFunc     func(error)
	HTTPHandler = func(*HTTPContext)
	WSHandler   = func(*WSContext)
	Next        = func()
	Redirect    = func(string)
)

const (
	RequestFinished = "RequestFinished"
	CatchException  = "CatchException"
)

type HTTPContext struct {
	*http.Request
	http.ResponseWriter

	dataWriter DataWriter

	body        Body
	form        Form
	file        File
	query       Query
	header      Header
	param       Param
	ParamKeys   map[string][]int
	ParamValues []string

	id  string
	typ string

	Next      Next
	Broker    broker.Broker
	Code      int
	Timestamp time.Time

	wsCfg     *websocket.Config
	wsConn    *websocket.Conn
	wsPayload WSPayload
}

const (
	HTTPType = "http"
	WSType   = "ws"
	RPCType  = "rpc"
	GQLType  = "gql"
)

func NewHTTPContext() *HTTPContext {
	return &HTTPContext{
		Code: http.StatusOK,
	}
}

func (c *HTTPContext) Init(w http.ResponseWriter, r *http.Request) {
	c.Timestamp = time.Now()
	c.ResponseWriter = w
	c.Request = r
	c.SetID()
}

func (c *HTTPContext) Reset() {
	c.Code = http.StatusOK
	c.typ = ""
	c.id = ""
	c.wsCfg = nil
	c.wsConn = nil
	c.wsPayload = nil
	c.body = nil
	c.form = nil
	c.file = nil
	c.query = nil
	c.header = nil
	c.param = nil
	c.ParamKeys = nil
	c.ParamValues = nil
	c.Next = nil
	c.ResponseWriter = nil
	c.Request = nil
	_ = c.Broker.Clear()
}

func (c *HTTPContext) SetType(t string) *HTTPContext {
	if c.typ == "" &&
		(t == HTTPType ||
			t == WSType ||
			t == RPCType ||
			t == GQLType) {
		c.typ = t
	}
	return c
}

func (c *HTTPContext) GetType() string {
	return c.typ
}

func (c *HTTPContext) SetID() {
	reqID := c.Header().Get(RequestID)
	if reqID == "" {
		reqID, _ = crypto.UUID()
	}

	if c.id == "" {
		c.id = reqID
	}
}

func (c *HTTPContext) GetID() string {
	return c.id
}
