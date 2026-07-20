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

// WSContext has no JSON/Status writer (that's HTTP-only) and the WS
// catch-chain (Broker-based) isn't wired yet (see common/exception_filter_ws.go),
// so this only logs and exercises the query-triggered panic.
func (instance LogModuleWSExceptionFilter) Catch(c ginject.WSContext, ex ginject.Exception) {
	fmt.Println("[Module] Log Module WS exception filter")

	if c.Conn.Config().Location.Query().Get("error_module_ws_ex") == "true" {
		panic(exception.InternalServerErrorException("LogModuleWSExceptionFilter error triggered"))
	}
}
