package context

import (
	"net/http"
	"strings"
	"time"

	"github.com/dangduoc08/gooh/utils"
)

type (
	Map      map[string]any
	ErrFn    func(error)
	Handler  = func(c *Context)
	Next     = func()
	Redirect = func(string)
)

type Responser interface {
	SetHeaders(map[string]string) Responser
	SetRoute(string) Responser
	GetRoute() string
	Status(int) Responser
	Redirect(string)
	Param() Values
	Text(string, ...any)
	JSONP(...any)
	JSON(...any)
}

type Context struct {
	*http.Request
	http.ResponseWriter

	dataWriter DataWriter

	param       Values
	ParamKeys   map[string][]int
	ParamValues []string

	route string

	Next      Next
	Event     *event
	Code      int
	Timestamp time.Time
}

func NewContext() *Context {
	return &Context{
		Code: http.StatusOK,
	}
}

func (c *Context) SetHeaders(pair map[string]string) Responser {
	for key, value := range pair {
		c.ResponseWriter.Header().Set(key, value)
	}
	return c
}

func (c *Context) Status(code int) Responser {
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

func (c *Context) SetRoute(route string) Responser {
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
	c.param = nil
	c.ParamKeys = nil
	c.ParamValues = nil
	c.Next = nil
	c.ResponseWriter = nil
	c.Request = nil
}
