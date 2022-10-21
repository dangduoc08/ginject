package gooh

import (
	"log"
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

type application struct {
	router *routing.Route
}

type Map map[string]interface{}

func Default() *application {
	app := application{
		routing.NewRoute(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		c := ctx.NewContext()
		isMatched, _, paramKeys, paramVals, handlers := app.router.Match(req.URL.Path, req.Method)
		isNext := true
		next := func() {
			isNext = true
		}
		c.ResponseWriter = w
		c.Request = req
		c.Next = next

		if isMatched {
			ctx.SetParamKeys(c, paramKeys)
			ctx.SetParamVals(c, paramVals)
			for _, handler := range handlers {
				if isNext {
					isNext = false
					handler(c)
				}
			}
		} else {
			// Invoke middlewares
			for _, middleware := range app.router.Middlewares {
				if isNext {
					isNext = false
					middleware(c)
				}
			}

			if isNext {
				c.Status(http.StatusNotFound)
				c.Event.Emit(ctx.REQUEST_FINISHED)
				http.NotFound(w, req)
			}
		}
	})

	return &app
}

func Route() *routing.Route {
	return routing.NewRoute()
}

func (app *application) ListenAndServe(addr string, handler http.Handler) error {
	app.router.Range(func(method, route string) {
		log.Default().Println("RouteExplorer", method, route)
	})

	return http.ListenAndServe(addr, handler)
}

func (app *application) Get(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Get(path, handlers...)
}

func (app *application) Head(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Head(path, handlers...)
}

func (app *application) Post(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Post(path, handlers...)
}

func (app *application) Put(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Put(path, handlers...)
}

func (app *application) Patch(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Patch(path, handlers...)
}

func (app *application) Delete(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Delete(path, handlers...)
}

func (app *application) Connect(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Connect(path, handlers...)
}

func (app *application) Options(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Options(path, handlers...)
}

func (app *application) Trace(path string, handlers ...ctx.Handler) routing.Router {
	return app.router.Trace(path, handlers...)
}

func (app *application) Group(prePath string, subRouters ...*routing.Route) routing.Router {
	return app.router.Group(prePath, subRouters...)
}

func (app *application) Use(handlers ...ctx.Handler) routing.Router {
	return app.router.Use(handlers...)
}

func (app *application) For(path string) func(handlers ...ctx.Handler) routing.Router {
	return app.router.For(path)
}
