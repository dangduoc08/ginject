package benchmarks

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogModuleHTTPGuard struct {
	common.Guard
	common.Logger
}

func (instance LogModuleHTTPGuard) CanActivate(c ginject.HTTPContext) bool {

	if c.Query().Get("error_module_http_guard") == "true" {
		panic(exception.InternalServerErrorException("LogModuleHTTPGuard error triggered"))
	}

	return true
}
