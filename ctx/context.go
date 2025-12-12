package ctx

import (
	"net/http"
	"strings"
	"time"

	"github.com/dangduoc08/ginject/utils"
)

type (
	Map      map[string]any
	ErrFn    func(error)
	Handler  = func(*Context)
	Next     = func()
	Redirect = func(string)
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

	route string
	ID    string
	Type  string

	Next      Next
	Event     *event
	Code      int
	Timestamp time.Time

	// Extend context
	// WebSocket
	WS *WS
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
	c.Event.Emit(REQUEST_FINISHED, c)
}

func (c *Context) JSON(data ...any) {
	c.dataWriter = &JSON{
		data:           data,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	c.Event.Emit(REQUEST_FINISHED, c)
}

func (c *Context) JSONP(data ...any) {
	callback := utils.StrRemoveSpace(c.URL.Query().Get("callback"))
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
	c.Event.Emit(REQUEST_FINISHED, c)
}

func (c *Context) GetRoute() string {
	return strings.Replace(c.route, "/["+c.Method+"]/", "", 1)
}

func (c *Context) SetRoute(route string) *Context {
	c.route = route
	return c
}

func (c *Context) Redirect(url string) {
	c.Status(http.StatusMovedPermanently)
	http.Redirect(c.ResponseWriter, c.Request, url, c.Code)
	c.Event.Emit(REQUEST_FINISHED, c)
}

func (c *Context) Reset() {
	c.Code = http.StatusOK
	c.route = ""
	c.Type = ""
	c.ID = ""
	c.WS = nil
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

func (c *Context) SetID(id string) *Context {
	if c.ID == "" {
		c.ID = id
	}
	return c
}

func (c *Context) GetID() string {
	return c.ID
}
