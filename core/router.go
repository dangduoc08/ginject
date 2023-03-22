package core

import (
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

func (app *App) Use(handlers ...ctx.Handler) *routing.Route {
	return app.route.Use(handlers...)
}

func (app *App) For(path string) func(handlers ...ctx.Handler) *routing.Route {
	return app.route.For(path)
}
