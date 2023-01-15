package core

import (
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

func (a *App) Use(handlers ...ctx.Handler) *routing.Route {
	return a.Route.Use(handlers...)
}

func (a *App) For(path string) func(handlers ...ctx.Handler) *routing.Route {
	return a.Route.For(path)
}
