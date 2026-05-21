package core

import (
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

/**
- Include default components
*/

var defaultException = exception.InternalServerErrorException("Unhandled exception has occurred")

type globalExceptionFilter struct{}

func (g globalExceptionFilter) Catch(c *ctx.Context, ex *exception.Exception) {
	code := ex.GetCode()
	if code == "" {
		code = defaultException.GetCode()
	}

	err := ex.Error()
	if err == "" {
		err = defaultException.Error()
	}
	data := ctx.Map{
		"code":  code,
		"error": err,
	}

	message := ex.GetResponse()
	var msgKind reflect.Kind
	if message != nil {
		msgKind = reflect.TypeOf(message).Kind()
	}
	switch msgKind {
	case reflect.String, reflect.Map, reflect.Slice, reflect.Struct:
		data["message"] = message
	default:
		data["message"] = defaultException.GetResponse()
	}

	httpCode, httpText := ex.GetHTTPStatus()
	if httpText == "" {
		httpCode, _ = defaultException.GetHTTPStatus()
	}

	switch c.GetType() {
	case ctx.HTTPType:
		c.Status(httpCode).JSON(data)
	case ctx.WSType:
		_ = c.WS.SendSelf(c, data)
	}
}
