package ctx

import (
	"net/http"
	"time"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/internal/str"
	"golang.org/x/net/websocket"
)

type (
	Map      map[string]any
	ErrFunc  func(error)
	Handler  = func(*Context)
	Next     = func()
	Redirect = func(string)
)

const (
	RequestFinished = "RequestFinished"
	CatchException  = "CatchException"
)

type Context struct {
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

	route      string
	cleanRoute string
	id         string
	Type       string

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

func NewContext() *Context {
	return &Context{
		Code: http.StatusOK,
	}
}

func (c *Context) Init(w http.ResponseWriter, r *http.Request) {
	c.Timestamp = time.Now()
	c.ResponseWriter = w
	c.Request = r
	c.SetID()
}

func (c *Context) Status(code int) *Context {
	c.Code = code
	return c
}

func (c *Context) Text(data string, args ...any) {
	c.dataWriter = &Text{
		data:           data,
		args:           args,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *Context) JSON(data ...any) {
	c.dataWriter = &JSON{
		data:           data,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *Context) JSONP(data ...any) {
	callback := str.RemoveSpace(c.URL.Query().Get("callback"))
	if callback == "" {
		c.JSON(data...)
		return
	}

	c.dataWriter = &JSONP{
		data:           data,
		responseWriter: c.ResponseWriter,
		callback:       callback,
	}
	c.dataWriter.WriteData(c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *Context) GetRoute() string {
	return c.cleanRoute
}

func (c *Context) SetRoute(route string) *Context {
	c.route = route
	suffix := "/[" + c.Method + "]/"
	c.cleanRoute = route[:len(route)-len(suffix)]
	return c
}

func (c *Context) Redirect(url string) {
	c.Status(http.StatusMovedPermanently)
	http.Redirect(c.ResponseWriter, c.Request, url, c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *Context) Reset() {
	c.Code = http.StatusOK
	c.route = ""
	c.cleanRoute = ""
	c.Type = ""
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

func (c *Context) SetType(t string) *Context {
	if c.Type == "" &&
		(t == HTTPType ||
			t == WSType ||
			t == RPCType ||
			t == GQLType) {
		c.Type = t
	}
	return c
}

func (c *Context) GetType() string {
	return c.Type
}

func (c *Context) SetID() {
	reqID := c.Header().Get(RequestID)
	if reqID == "" {
		reqID, _ = crypto.UUID()
	}

	if c.id == "" {
		c.id = reqID
	}
}

func (c *Context) GetID() string {
	return c.id
}

func (c *Context) SetWSConfig(wsCfg *websocket.Config) {
	c.wsCfg = wsCfg
}

func (c *Context) GetWSConfig() *websocket.Config {
	return c.wsCfg
}

func (c *Context) SetWSConn(conn *websocket.Conn) *Context {
	c.wsConn = conn
	return c
}

func (c *Context) WSConn() *websocket.Conn {
	return c.wsConn
}

func (c *Context) SetWSPayload(p WSPayload) *Context {
	c.wsPayload = p
	return c
}

func (c *Context) WSPayload() WSPayload {
	return c.wsPayload
}
