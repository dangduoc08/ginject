package ctx

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Response interface {
	Set(fields map[string]string) Response // Set HTTP header field to value
	Type(contentType string) Response      // Set HTTP header content-type
	// Send(statusCode int, content string, arguments ...interface{}) // Response text or HTML
	JSON(statusCode int, datas ...interface{}) // Response JSON
	// JSONP(statusCode int, datas ...interface{})                    // Response JSONP
	// Error(callback func(interface{}))                              // Response Error
}

type Context struct {
	Req    *http.Request
	Res    http.ResponseWriter
	Event  *Event
	Route  *Route
	Params Param[interface{}]
	Next   func()
}

// Set response HTTP headers
// by implement Header.().Add() method of http.ResponseWriter interface
func (ctx *Context) Set(fields map[string]string) Response {
	for field, value := range fields {
		ctx.Res.Header().Add(field, value)
	}

	return ctx
}

// Set content-type header
// by implement Header.().Set() method of http.ResponseWriter interface
func (ctx *Context) Type(contentType string) Response {
	ctx.Res.Header().Set("Content-Type", contentType)

	return ctx
}

// JSON to reponse JSON type
func (ctx *Context) JSON(statusCode int, datas ...interface{}) {

	// Datas can be string, struct or map[string]interface{}
	data := datas[0]

	switch data.(type) {

	// Handle case datas are string
	case string:

		// Format string with params
		fStr := fmt.Sprintf(data.(string), datas[1:]...)

		// Parse string to raw JSON
		data = json.RawMessage(fStr)
	default:

		// If datas are not string
		// only accept one argument
		if len(datas) > 1 {
			panic("JSON use map or struct type only accepts a agrument")
		}
	}

	// Parse to JSON
	buffer, err := json.Marshal(&data)

	if err != nil {
		panic(err.Error())
	}

	ctx.Type("application/json")
	ctx.Res.WriteHeader(statusCode)
	ctx.Res.Write(buffer)
	ctx.Event.Emit("finish")
	ctx.Next()
}

// Send string or HTML string
// second params are variable value
func (ctx *Context) Send(statusCode int, content string, arguments ...interface{}) {
	ctx.Res.WriteHeader(statusCode)
	fmt.Fprintf(ctx.Res, content, arguments...)
	ctx.Event.Emit("finish")
	ctx.Next()
}
