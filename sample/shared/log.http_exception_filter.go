package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

type LogHTTPExceptionFilter struct {
	common.ExceptionFilter
	common.Logger
}

func (instance LogHTTPExceptionFilter) Catch(c ginject.HTTPContext, ex ginject.Exception) {
	fmt.Println("[Global] Log HTTP exception filter")

	if c.Query().Get("error_http_ex") == "true" {
		panic(exception.InternalServerErrorException("LogHTTPExceptionFilter error triggered"))
	}

	httpCode, _ := ex.GetHTTPStatus()
	c.Status(httpCode).JSON(ctx.Map{
		"code":    ex.GetCode(),
		"error":   ex.Error(),
		"message": ex.GetResponse(),
	})
}
