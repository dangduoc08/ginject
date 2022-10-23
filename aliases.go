package gooh

import (
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

type (
	Context = *ctx.Context
	App     = *core.App
	Map     ctx.Map
	Route   *routing.Route
	Handler ctx.Handler
)
