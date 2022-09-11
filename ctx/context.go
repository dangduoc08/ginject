package ctx

import (
	"context"
	"net/http"
)

type Context struct {
	context.Context
	Req    *http.Request
	Res    http.ResponseWriter
	Params Param[interface{}]
}
