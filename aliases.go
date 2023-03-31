package gooh

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/routing"
)

type (
	Context = *context.Context
	App     = *core.App
	Map     context.Map
	Route   *routing.Route
)
