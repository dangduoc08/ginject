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
	c.Status(httpCode).JSON(context.Map{
		"code":    internalServerErrorException.GetCode(),
		"error":   internalServerErrorException.Error(),
		"message": internalServerErrorException.GetResponse(),
	})
}
