package core

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/exception"
)

// includes default components

type GlobalExceptionFilter struct{}

func (g GlobalExceptionFilter) Catch(c *context.Context, e *exception.HTTPException) {
	internalServerErrorException := exception.InternalServerErrorException("Unhandled exception has occurred")
	httpCode, _ := internalServerErrorException.GetHTTPStatus()
	c.Status(httpCode).JSON(context.Map{
		"code":    internalServerErrorException.GetCode(),
		"error":   internalServerErrorException.Error(),
		"message": internalServerErrorException.GetResponse(),
	})
}
