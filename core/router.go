package core

import (
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

func (a *App) Get(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Get(path, handlers...)
}

func (a *App) Head(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Head(path, handlers...)
}

func (a *App) Post(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Post(path, handlers...)
}

func (a *App) Put(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Put(path, handlers...)
}

func (a *App) Patch(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Patch(path, handlers...)
}

func (a *App) Delete(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Delete(path, handlers...)
}

func (a *App) Connect(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Connect(path, handlers...)
}

func (a *App) Options(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Options(path, handlers...)
}

func (a *App) Trace(path string, handlers ...ctx.Handler) *routing.Route {
	return a.Route.Trace(path, handlers...)
}

func (a *App) Group(prePath string, subRouters ...*routing.Route) *routing.Route {
	return a.Route.Group(prePath, subRouters...)
}

func (a *App) Use(handlers ...ctx.Handler) *routing.Route {
	return a.Route.Use(handlers...)
}

func (a *App) For(path string) func(handlers ...ctx.Handler) *routing.Route {
	return a.Route.For(path)
}
