package context

import (
	goCtx "context"
	"net/http"
)

type Context struct {
	goCtx.Context
	Req    *http.Request
	Res    ResponseExtender
	Params Param[interface{}]
}
