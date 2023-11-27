package core

import (
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

	messages := ex.GetResponse()
	if messages == "" {
		messages = internalServerErrorException.GetResponse()
	}

	httpCode, httpText := ex.GetHTTPStatus()
	if httpText == "" {
		httpCode, _ = internalServerErrorException.GetHTTPStatus()
	}

	data := context.Map{
		"code":     code,
		"error":    err,
		"messages": messages,
	}
	requestType := c.GetType()
	if requestType == context.HTTPType {
		c.Status(httpCode).JSON(data)
	} else if requestType == context.WSType {
		c.WS.SendSelf(c, data)
	}
}
