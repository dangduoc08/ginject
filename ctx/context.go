package ctx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Handler func(c *Context)
type ErrFn func(interface{})

type Responser interface {
	Set(pair map[string]string) Responser
	Status(code int) Responser
	Text(content string, args ...interface{})
	JSON(args ...interface{})
	Param() Values
	// JSONP(args ...interface{})
	// Error(ErrFn)
}

type Context struct {
	*http.Request
	http.ResponseWriter

	param     Values
	paramKeys map[string][]int
	paramVals []string

	Next      func()
	Event     *event
	Code      int
	Timestamp time.Time
}

func NewContext() *Context {
	return &Context{
		Code:      http.StatusOK,
		Timestamp: time.Now(),
		Event:     newEvent(),
	}
}

func SetParamKeys(c *Context, paramKeys map[string][]int) {
	c.paramKeys = paramKeys
}

func SetParamVals(c *Context, paramVals []string) {
	c.paramVals = paramVals
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

func (c *Context) Text(content string, args ...interface{}) {
	c.WriteHeader(c.Code)
	fmt.Fprintf(c.ResponseWriter, content, args...)
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
	c.WriteHeader(c.Code)
	c.ResponseWriter.Write(buf)
	c.Event.Emit(REQUEST_FINISHED)
}
