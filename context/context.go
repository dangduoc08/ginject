package context

import (
	"net/http"
	"time"

	"github.com/dangduoc08/gooh/utils"
)

type (
	Map     map[string]any
	ErrFn   func(error)
	Handler = func(c *Context)
	Next    = func()
)

type Responser interface {
	Set(map[string]string) Responser
	Status(int) Responser
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

func (c *Context) Set(pair map[string]string) Responser {
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

func (c *Context) Reset() {
	c.Code = http.StatusOK
	c.param = nil
	c.ParamKeys = nil
	c.ParamValues = nil
	c.Next = nil
	c.ResponseWriter = nil
	c.Request = nil
}
