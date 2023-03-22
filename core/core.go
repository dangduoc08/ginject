package core

import (
	"log"
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type App struct {
	route  *routing.Route
	module *Module
}

func New() *App {
	app := App{
		route: routing.NewRoute(),
	}
	event := ctx.NewEvent()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		c := ctx.NewContext()
		c.Event = event
		isNext := true
		c.ResponseWriter = w
		c.Request = req
		c.Next = func() {
			isNext = true
		}

		defer func() {
			if rec := recover(); rec != nil {
				c.Event.Emit(ctx.REQUEST_FAILED, rec)
			}
		}()

		isMatched, _, paramKeys, paramVals, handlers := app.route.Match(req.URL.Path, req.Method)

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
			for _, middleware := range app.route.Middlewares {
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

func (app *App) Create(m *Module) {
	app.module = m.Inject()
	app.route.Group("/", app.module.router)
}

func (app *App) ListenAndServe(addr string, handler http.Handler) error {
	app.route.Range(func(method, route string) {
		log.Default().Println("RouteExplorer", method, route)
	})

	return http.ListenAndServe(addr, handler)
}

func (app *App) Get(p Provider) any {
	findKey := genProviderKey(p)
	return utils.ArrFind(app.module.providers, func(provider Provider, i int) bool {
		return genProviderKey(provider) == findKey
	})
}
