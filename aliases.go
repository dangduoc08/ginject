package gogo

import (
	"net/http"

	"github.com/dangduoc08/gogo/aggregation"
	"github.com/dangduoc08/gogo/core"
	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/exception"
	"github.com/dangduoc08/gogo/routing"
	"github.com/dangduoc08/gogo/versioning"
)

type (
	App         = *core.App
	Map         = ctx.Map
	Router      = *routing.Router
	Aggregation = *aggregation.Aggregation
	Exception   = *exception.Exception
	Versioning  = versioning.Versioning
	FieldLevel  = ctx.FieldLevel

	// decorators
	Context   = *ctx.Context
	Request   = *http.Request
	Response  = http.ResponseWriter
	Body      = ctx.Body
	Form      = ctx.Form
	File      = ctx.File
	Query     = ctx.Query
	Header    = ctx.Header
	Param     = ctx.Param
	WSPayload = ctx.WSPayload
	Next      = ctx.Next
	Redirect  = ctx.Redirect
)
