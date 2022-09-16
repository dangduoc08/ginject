package ctx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Handler func(c *Context)

type Responser interface {
	Set(pair map[string]string) Responser
	Status(code int) Responser
	Text(content string, args ...interface{})
	JSON(args ...interface{})
	Param() Values
	// JSONP(args ...interface{})
	// Error(callback func(interface{}))
}

type Context struct {
	req    *http.Request
	writer http.ResponseWriter

	Query     func() url.Values
	URL       *url.URL
	UserAgent func() string
	Method    string

	Next      func()
	Event     *event
	Code      int
	Timestamp time.Time
	ParamKeys map[string][]int
	ParamVals []string
}

func NewContext() *Context {
	return &Context{
		Code:      http.StatusOK,
		Timestamp: time.Now(),
		Event:     newEvent(),
	}
}

func SetReq(c *Context, req *http.Request) {
	c.req = req
}

func SetRes(c *Context, writer http.ResponseWriter) {
	c.writer = writer
}

func (c *Context) Set(pair map[string]string) Responser {
	for key, value := range pair {
		c.writer.Header().Set(key, value)
	}

	return c
}

func (c *Context) Status(code int) Responser {
	c.Code = code

	return c
}

func (c *Context) Text(content string, args ...interface{}) {
	c.writer.WriteHeader(c.Code)
	fmt.Fprintf(c.writer, content, args...)
	c.Event.Emit(REQUEST_FINISHED)
}

func (c *Context) JSON(args ...interface{}) {
	data := args[0]

	switch args[0].(type) {
	case string:
		str := fmt.Sprintf(data.(string), args[1:]...)
		data = json.RawMessage(str)
	}

	buf, err := json.Marshal(&data)

	if err != nil {
		panic(err.Error())
	}

	c.Set(map[string]string{
		"Content-Type": "application/json",
	})
	c.writer.WriteHeader(c.Code)
	c.writer.Write(buf)
	c.Event.Emit(REQUEST_FINISHED)
}
