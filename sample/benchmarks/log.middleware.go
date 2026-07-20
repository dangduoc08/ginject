package benchmarks

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogModuleMiddleware struct {
	common.Middleware
	common.Logger
}

func (instance LogModuleMiddleware) Use(r *http.Request, w http.ResponseWriter, next ginject.Next) {
	fmt.Println("[Module] Pre middleware")

	if r.URL.Query().Get("error_module_mw") == "true" {
		panic(exception.InternalServerErrorException("LogModuleMiddleware error triggered"))
	}

	next()
	fmt.Println("[Module] Post middleware")
}
