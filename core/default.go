package core

import (
	"reflect"
	"strings"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/exception"
)

/**
- Include default components
*/

type globalExceptionFilter struct{}

func (g globalExceptionFilter) Catch(c *context.Context, ex *exception.HTTPException) {
	internalServerErrorException := exception.InternalServerErrorException("Unhandled exception has occurred")

	code := ex.GetCode()
	if code == "" {
		code = internalServerErrorException.GetCode()
	}

	err := ex.Error()
	if err == "" {
		err = internalServerErrorException.Error()
	}
	data := context.Map{
		"code":  code,
		"error": err,
	}

	message := ex.GetResponse()
	switch reflect.TypeOf(message).Kind() {
	case reflect.String:
		data["message"] = message
	case reflect.Slice:
		if messages, ok := message.([]string); ok {
			data["message"] = strings.Join(messages, ", ")
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
	if requestType == context.HTTPType {
		c.Status(httpCode).JSON(data)
	} else if requestType == context.WSType {
		c.WS.SendSelf(c, data)
	}
}
