package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogHTTPGuard struct {
	common.Guard
	common.Logger
}

func (instance LogHTTPGuard) CanActivate(c ginject.HTTPContext) bool {
	fmt.Println("[Global] Log HTTP guard")

	if c.Query().Get("error_http_guard") == "true" {
		panic(exception.InternalServerErrorException("LogHTTPGuard error triggered"))
	}

	return true
}
