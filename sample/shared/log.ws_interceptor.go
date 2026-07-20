package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogWSInterceptor struct {
	common.Logger
}

func (instance LogWSInterceptor) Intercept(c ginject.WSContext, aggregation ginject.Aggregation) any {
	fmt.Println("[Global] Pre WS interceptor")

	if c.Conn.Config().Location.Query().Get("error_ws_pre_intercept") == "true" {
		panic(exception.InternalServerErrorException("LogWSInterceptor error triggered"))
	}

	return aggregation.Pipe()
}
