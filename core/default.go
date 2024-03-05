package core

import (
	"reflect"
	"strings"

	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/exception"
)

/**
- Include default components
*/

type globalExceptionFilter struct{}

func (g globalExceptionFilter) Catch(c *ctx.Context, ex *exception.HTTPException) {
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
	case reflect.String:
		data["message"] = message
	case reflect.Slice:
		if messageArr, ok := message.([]string); ok {
			data["message"] = strings.Join(messageArr, ", ")
		} else if messageObj, ok := message.([]map[string]any); ok {
			data["messages"] = messageObj
		} else {
			data["message"] = internalServerErrorException.GetResponse()
		}
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
