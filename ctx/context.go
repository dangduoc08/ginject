package ctx

import (
	"context"
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

	// Exec is the runtime control plane for this request or WS connection.
	// Set by the framework entrypoint; nil-safe via GetExec().
	Exec *ExecutionContext

	// WebSocket state — legacy, kept for backward compatibility.
	WS *WS

	// WSCtx is the new, lifecycle-managed WebSocket context.
	// Populated only for WebSocket connections.
	WSCtx *WSContext
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

// GetExec returns the ExecutionContext for this request.
// Nil-safe: if Exec isn't wired yet a shim is created from r.Context()
// (or context.Background() when Request is absent) and cached on the struct.
func (c *Context) GetExec() *ExecutionContext {
	if c.Exec != nil {
		return c.Exec
	}
	parent := context.Background()
	if c.Request != nil {
		parent = c.Request.Context()
	}
	exec, _ := NewHTTPExecutionContext(parent, 30*time.Second)
	c.Exec = exec
	return c.Exec
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
	c.WSCtx = nil
	c.Exec = nil // cancel is deferred by the entrypoint before Reset is called
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
