package shared

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogMiddleware struct {
	common.Middleware
	common.Logger
}

func (instance LogMiddleware) Use(r *http.Request, w http.ResponseWriter, next ginject.Next) {
	fmt.Println("[Global] Pre middleware")

	if r.URL.Query().Get("error_mw") == "true" {
		panic(exception.InternalServerErrorException("LogMiddleware error triggered"))
	}

	next()
	fmt.Println("[Global] Post middleware")
}
