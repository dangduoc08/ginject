package benchmarks

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogModuleWSExceptionFilter struct {
	common.ExceptionFilter
	common.Logger
}

func (instance LogModuleWSExceptionFilter) Catch(c ginject.WSContext, ex ginject.Exception) {
	fmt.Println("[Module] Log Module WS exception filter")

	if c.Conn.Config().Location.Query().Get("error_module_ws_ex") == "true" {
		panic(exception.InternalServerErrorException("LogModuleWSExceptionFilter error triggered"))
	}

	c.Send(ginject.Map{
		"code":    ex.GetCode(),
		"error":   ex.Error(),
		"message": ex.GetMessage(),
	})
}
