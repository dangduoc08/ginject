package core

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type App struct {
	route  *routing.Route
	module *Module
	pool   sync.Pool
}

func New() *App {
	event := context.NewEvent()

	app := App{
		route: routing.NewRoute(),
		pool: sync.Pool{
			New: func() any {
				c := context.NewContext()
				c.Event = event

				return c
			},
		},
	}

	return &app
}

func (app *App) Create(m *Module) {
	app.module = m.Inject()
	app.route.Group("/", app.module.router)
}

func (app *App) Get(p Provider) any {
	k := genProviderKey(p)
	return utils.ArrFind(app.module.providers, func(provider Provider, i int) bool {
		return genProviderKey(provider) == k
	})
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := app.pool.Get().(*context.Context)
	c.Timestamp = time.Now()
	defer app.pool.Put(c)
	app.handleRequest(w, r, c)
	c.Reset()
}

func (app *App) ListenAndServe(addr string) error {
	app.route.Range(func(method, route string) {
		log.Default().Println("RouteExplorer", method, route)
	})

	return http.ListenAndServe(addr, app)
}

func (app *App) handleRequest(w http.ResponseWriter, r *http.Request, c *context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			c.Event.Emit(context.REQUEST_FAILED, c, rec)
		}
	}()

	isNext := true
	c.ResponseWriter = w
	c.Request = r
	c.Next = func() {
		isNext = true
	}

	isMatched, _, paramKeys, paramValues, handlers := app.route.Match(r.URL.Path, r.Method)

	if isMatched {
		c.ParamKeys = paramKeys
		c.ParamValues = paramValues
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
			c.Event.Emit(context.REQUEST_FINISHED, c)
			http.NotFound(w, r)
		}
	}
}
