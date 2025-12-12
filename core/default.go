package core

import (
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

/**
- Include default components
*/

type globalExceptionFilter struct{}

func (g globalExceptionFilter) Catch(c *ctx.Context, ex *exception.Exception) {
	internalServerErrorException := exception.InternalServerErrorException("Unhandled exception has occurred")

	code := ex.GetCode()
	if code == "" {
		code = internalServerErrorException.GetCode()
	}

	err := ex.Error()
	if err == "" {
		err = internalServerErrorException.Error()
	}
	data := ctx.Map{
		"code":  code,
		"error": err,
	}

	message := ex.GetResponse()
	switch reflect.TypeOf(message).Kind() {
	case reflect.String, reflect.Map, reflect.Slice, reflect.Struct:
		data["message"] = message
	default:
		data["message"] = internalServerErrorException.GetResponse()
	}

	httpCode, httpText := ex.GetHTTPStatus()
	if httpText == "" {
		httpCode, _ = internalServerErrorException.GetHTTPStatus()
	}

	requestType := c.GetType()
	if requestType == ctx.HTTPType {
		c.Status(httpCode).JSON(data)
	} else if requestType == ctx.WSType {
		c.WS.SendSelf(c, data)
	}
}
