package core

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
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

// link to aliases
var (
	CONTEXT  = "*context.Context"
	REQUEST  = "*http.Request"
	RESPONSE = "http.ResponseWriter"
	PARAM    = "context.Values"
	QUERY    = "url.Values"
	HEADER   = "http.Header"
	NEXT     = "func()"

	dependencies = map[string]int{
		CONTEXT:  1,
		REQUEST:  1,
		RESPONSE: 1,
		PARAM:    1,
		QUERY:    1,
		HEADER:   1,
		NEXT:     1,
	}
)

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

	isMatched, matchedRoute, paramKeys, paramValues, handlers := app.route.Match(r.URL.Path, r.Method)

	if isMatched {
		c.ParamKeys = paramKeys
		c.ParamValues = paramValues
		for _, handler := range handlers {
			if isNext {
				isNext = false
				if handler == nil {

					// handler = nil
					// meaning this is injectable handler
					injectableHandler := app.route.InjectableHandlers[matchedRoute]
					app.provideAndInvoke(injectableHandler, c)
				} else {
					handler(c)
				}
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

func (app *App) provideAndInvoke(f any, c *context.Context) {
	args := []reflect.Value{}
	injectableFnType := reflect.TypeOf(f)

	for i := 0; i < injectableFnType.NumIn(); i++ {

		// get the type of the current input parameter
		dynamicArgKey := injectableFnType.In(i).String()
		if _, ok := dependencies[dynamicArgKey]; ok {
			args = append(args, reflect.ValueOf(app.getDependency(dynamicArgKey, c)))
		} else {
			panic(fmt.Errorf(
				"can't resolve dependencies of the handler. Please make sure that the argument dependency at index [%v] is available in the handler",
				i,
			))
		}
	}

	// invoke handler
	reflect.ValueOf(f).Call(args)
}

func (app *App) getDependency(k string, c *context.Context) any {
	switch k {
	case CONTEXT:
		return c
	case REQUEST:
		return c.Request
	case RESPONSE:
		return c.ResponseWriter
	case PARAM:
		return c.Param()
	case QUERY:
		return c.URL.Query()
	case HEADER:
		return c.Request.Header
	case NEXT:
		return c.Next
	}

	return dependencies
}
