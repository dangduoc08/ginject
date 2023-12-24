package gooh

import (
	"net/http"

	"github.com/dangduoc08/gooh/aggregation"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/routing"
)

type (
	App           = *core.App
	Map           = ctx.Map
	Router        = *routing.Router
	Aggregation   = *aggregation.Aggregation
	HTTPException = *exception.HTTPException

	// decorators
	Context    = *ctx.Context
	Request    = *http.Request
	Response   = http.ResponseWriter
	Body       = ctx.Body
	Form       = ctx.Form
	File       = ctx.File
	Query      = ctx.Query
	Header     = ctx.Header
	Param      = ctx.Param
	WSPayload  = ctx.WSPayload
	Next       = ctx.Next
	Redirect   = ctx.Redirect
	FieldLevel = ctx.FieldLevel
)
