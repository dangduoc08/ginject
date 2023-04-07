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
	REDIRECT = "func(string)"

	dependencies = map[string]int{
		CONTEXT:  1,
		REQUEST:  1,
		RESPONSE: 1,
		PARAM:    1,
		QUERY:    1,
		HEADER:   1,
		NEXT:     1,
		REDIRECT: 1,
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
		c.SetRoute(matchedRoute)
		c.ParamKeys = paramKeys
		c.ParamValues = paramValues
		if r.Method == http.MethodPost {
			c.Status(http.StatusCreated)
		}

		for _, handler := range handlers {
			if isNext {
				isNext = false
				if handler == nil {

					// handler = nil / main handler
					// meaning this is injectable handler
					injectableHandler := app.route.InjectableHandlers[matchedRoute]
					values := app.provideAndInvoke(injectableHandler, c)
					if len(values) == 1 {
						app.selectData(c, values[0])
					} else if len(values) > 1 {
						app.selectStatusCode(c, values[0])
						app.selectData(c, values[1])
					}
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

func (app *App) provideAndInvoke(f any, c *context.Context) []reflect.Value {
	args := []reflect.Value{}
	getFnArgs(f, func(dynamicArgKey string, i int) {
		if _, ok := dependencies[dynamicArgKey]; ok {
			args = append(args, reflect.ValueOf(app.getDependency(dynamicArgKey, c)))
		} else {
			panic(fmt.Errorf(
				"can't resolve dependencies of the %v. Please make sure that the argument dependency at index [%v] is available in the handler",
				reflect.TypeOf(f).String(),
				i,
			))
		}
	})

	return reflect.ValueOf(f).Call(args)
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
	case REDIRECT:
		return c.Redirect
	}

	return dependencies
}

func (app *App) selectData(c *context.Context, data reflect.Value) {
	switch data.Type().Kind() {
	case
		reflect.Map,
		reflect.Slice,
		reflect.Struct,
		reflect.Interface:
		c.JSON(data.Interface())
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		c.Text(fmt.Sprint(data))
	case
		reflect.Pointer,
		reflect.UnsafePointer:
		c.Text(fmt.Sprint(data.UnsafePointer()))
	case
		reflect.String:
		c.Text(data.Interface().(string))
	case
		reflect.Func:
		c.Text(data.Type().String())
	}
}

func (app *App) selectStatusCode(c *context.Context, statusCode reflect.Value) {
	statusCodeKind := statusCode.Type().Kind()

	if statusCodeKind == reflect.Int {
		status := int(statusCode.Int())
		if http.StatusText(status) != "" {
			c.Status(status)
		}
	} else if statusCodeKind == reflect.Interface {
		if status, ok := statusCode.Interface().(int); ok &&
			http.StatusText(status) != "" {
			c.Status(status)
		}
	}
}
