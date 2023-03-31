package core

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
)

func (app *App) Use(handlers ...context.Handler) *routing.Route {
	return app.route.Use(handlers...)
}

func (app *App) For(path string) func(handlers ...context.Handler) *routing.Route {
	return app.route.For(path)
}
