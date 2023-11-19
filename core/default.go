package core

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/exception"
)

/**
- Include default components
*/

type globalExceptionFilter struct{}

func (g globalExceptionFilter) Catch(c *context.Context, e *exception.HTTPException) {
	internalServerErrorException := exception.InternalServerErrorException("Unhandled exception has occurred")
	httpCode, _ := internalServerErrorException.GetHTTPStatus()
	data := context.Map{
		"code":    internalServerErrorException.GetCode(),
		"error":   internalServerErrorException.Error(),
		"message": internalServerErrorException.GetResponse(),
	}
	requestType := c.GetType()
	if requestType == context.HTTPType {
		c.Status(httpCode).JSON(data)
	} else if requestType == context.WSType {
		c.WS.SendSelf(c, data)
	}
}
