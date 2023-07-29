package gooh

import (
	"net/http"

	"github.com/dangduoc08/gooh/aggregation"
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/routing"
)

type (
	App           = *core.App
	Map           = context.Map
	Router        = *routing.Router
	Aggregation   = *aggregation.Aggregation
	HTTPException = *exception.HTTPException

	// decorators
	Context  = *context.Context
	Request  = *http.Request
	Response = http.ResponseWriter
	Body     = context.Body
	Query    = context.Query
	Header   = context.Header
	Param    = context.Param
	Next     = context.Next
	Redirect = context.Redirect
)
