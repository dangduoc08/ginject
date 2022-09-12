package gooh

import (
	"log"
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

type application struct {
	router *routing.Router
}

func Default() *application {
	appInstance := application{
		routing.NewRouter(),
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		isMatched, matchedRoute, routerData := appInstance.router.Match(req.Method, req.URL.Path)
		isNext := true
		next := func() {
			isNext = true
		}

		ctx := ctx.Context{
			Req:    req,
			Res:    writer,
			Params: *routerData.Params,
			Event:  ctx.NewEvent(),
			Route: &ctx.Route{
				Path: matchedRoute,
			},
			Next: next,
		}

		if isMatched {
			for _, handler := range *routerData.Handlers {
				if isNext {
					isNext = false
					handler(&ctx)
				}
			}
		} else {
			http.NotFound(writer, req)
		}
	})

	return &appInstance
}

func Router() *routing.Router {
	return routing.NewRouter()
}

func (appInstance *application) ListenAndServe(addr string, handler http.Handler) error {
	for _, el := range appInstance.router.RouteMapDataArr {
		for route := range el {
			log.Default().Println("RouteExplorer", route)
		}
	}

	return http.ListenAndServe(addr, handler)
}

func (appInstance *application) Get(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Get(route, handlers...)
}

func (appInstance *application) Head(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Head(route, handlers...)
}

func (appInstance *application) Post(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Post(route, handlers...)
}

func (appInstance *application) Put(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Put(route, handlers...)
}

func (appInstance *application) Patch(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Patch(route, handlers...)
}

func (appInstance *application) Delete(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Delete(route, handlers...)
}

func (appInstance *application) Connect(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Connect(route, handlers...)
}

func (appInstance *application) Options(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Options(route, handlers...)
}

func (appInstance *application) Trace(route string, handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Trace(route, handlers...)
}

func (appInstance *application) Group(prefixRoute string, subRouters ...*routing.Router) routing.Routable {
	return appInstance.router.Group(prefixRoute, subRouters...)
}

func (appInstance *application) Use(handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.Use(handlers...)
}

func (appInstance *application) For(route string) func(handlers ...ctx.Handler) routing.Routable {
	return appInstance.router.For(route)
}
