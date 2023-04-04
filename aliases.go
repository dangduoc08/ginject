package gooh

import (
	"net/http"
	"net/url"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/routing"
)

type (
	App   = *core.App
	Map   = context.Map
	Route = *routing.Route

	// decorators
	Context  = *context.Context
	Request  = *http.Request
	Response = http.ResponseWriter
	Param    = context.Values
	Query    = url.Values
	Header   = http.Header
	Next     = context.Next
)
