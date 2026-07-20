package benchmarks

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogModuleWSInterceptor struct {
	common.Logger
}

func (instance LogModuleWSInterceptor) Intercept(c ginject.WSContext, aggregation ginject.Aggregation) any {
	fmt.Println("[Module] Pre Module WS interceptor")

	if c.Conn.Config().Location.Query().Get("error_module_ws_pre_intercept") == "true" {
		panic(exception.InternalServerErrorException("LogModuleWSInterceptor error triggered"))
	}

	return aggregation.Pipe()
}
