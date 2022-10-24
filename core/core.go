package core

import (
	"log"
	"net/http"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
)

type App struct {
	Route *routing.Route
}

func New() *App {
	a := App{
		routing.NewRoute(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		c := ctx.NewContext()
		defer func() {
			if rec := recover(); rec != nil {
				c.Status(http.StatusInternalServerError)
				c.Event.Emit(ctx.REQUEST_FINISHED)
				http.Error(w, rec.(error).Error(), http.StatusInternalServerError)
			}
		}()

		isMatched, _, paramKeys, paramVals, handlers := a.Route.Match(req.URL.Path, req.Method)
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
			for _, middleware := range a.Route.Middlewares {
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

	return &a
}

func (a *App) Create(m *common.Module) {
	mainModule := m.GetInstance()
	a.Group("/", mainModule.ModuleRouter)
}

func (a *App) ListenAndServe(addr string, handler http.Handler) error {
	a.Route.Range(func(method, route string) {
		log.Default().Println("RouteExplorer", method, route)
	})

	return http.ListenAndServe(addr, handler)
}
