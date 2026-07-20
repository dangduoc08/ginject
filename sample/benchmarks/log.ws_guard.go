package benchmarks

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogModuleWSGuard struct {
	common.Guard
	common.Logger
}

func (instance LogModuleWSGuard) CanActivate(c ginject.WSContext) bool {
	fmt.Println("[Module] Log Module WS guard")

	if c.Conn.Config().Location.Query().Get("error_module_ws_guard") == "true" {
		panic(exception.InternalServerErrorException("LogModuleWSGuard error triggered"))
	}

	return true
}
