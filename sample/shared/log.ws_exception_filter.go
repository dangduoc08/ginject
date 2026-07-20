package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogWSExceptionFilter struct {
	common.ExceptionFilter
	common.Logger
}

func (instance LogWSExceptionFilter) Catch(c ginject.WSContext, ex ginject.Exception) {
	fmt.Println("[Global] Log WS exception filter")

	if c.Conn.Config().Location.Query().Get("error_ws_ex") == "true" {
		panic(exception.InternalServerErrorException("LogWSExceptionFilter error triggered"))
	}

	c.Send(ginject.Map{
		"code":    ex.GetCode(),
		"error":   ex.Error(),
		"message": ex.GetMessage(),
	})
}
