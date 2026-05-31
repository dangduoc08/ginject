package core

import (
	stdHTTP "net/http"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/utils"
	"github.com/dangduoc08/ginject/versioning"
)

type HTTP struct {
	route *routing.Router

	versioning         *versioning.Versioning
	isEnableVersioning bool

	invokeHandler func(f any, c *ctx.Context) []reflect.Value

	ctxPool                                sync.Pool
	catchRESTFnsMap                        map[string][]common.Catch
	serveStaticMapToLastWildcardSlashIndex map[string]int // to check public dir URL if has * at last
}

func newHTTP() *HTTP {
	http := HTTP{
		route:                                  routing.NewRouter(),
		catchRESTFnsMap:                        make(map[string][]common.Catch),
		serveStaticMapToLastWildcardSlashIndex: make(map[string]int),
		ctxPool: sync.Pool{
			New: func() any {
				c := ctx.NewContext()
				c.Event = ctx.NewEvent()
				return c
			},
		},
	}

	return &http
}

func (http *HTTP) ServeHTTP(w stdHTTP.ResponseWriter, r *stdHTTP.Request) {
	c := http.ctxPool.Get().(*ctx.Context)
	c.Timestamp = time.Now()
	c.ResponseWriter = w
	c.Request = r
	ctxID := http.getContextID(c)
	c.SetID(ctxID)

	defer func() {
		c.Reset()
		http.ctxPool.Put(c)
	}()

	if r.URL.Path == "/ws" || r.URL.Path == "/ws/" {
		// c.SetType(ctx.WSType)
		// websocket.Handler.ServeHTTP(func(wsConn *websocket.Conn) {
		// 	app.handleWSRequest(wsConn, w, r, c)
		// }, w, r)
	} else {
		c.SetType(ctx.HTTPType)
		c.ResponseWriter.Header().Set(ctx.REQUEST_ID, c.GetID())

		http.handleRESTRequest(c)
	}
}

func (http *HTTP) enableVersioning(v versioning.Versioning) {
	if v.Key == "" {
		v.Key = "v"
	}
	http.versioning = &v
	http.isEnableVersioning = true
}

func (http *HTTP) addMainHandler(moduleHandler common.RESTLayer) {
	httpMethod := routing.OperationsMapHTTPMethods[moduleHandler.Method]
	if moduleHandler.Method == routing.SERVE {
		r := moduleHandler.Route
		lr := len(r)
		lastWildcardSlashIndex := 0 // zero mean use config dir
		if lr >= 2 && r[lr-2:] == "*/" {
			lastWildcardSlashIndex = strings.Count(r, "/") - 1
		}

		http.serveStaticMapToLastWildcardSlashIndex[routing.MethodRouteVersionToPattern(httpMethod, moduleHandler.Route, moduleHandler.Version)] = lastWildcardSlashIndex
	}
	http.route.AddInjectableHandler(httpMethod, moduleHandler.Route, moduleHandler.Version, moduleHandler.Handler)
}

func (http *HTTP) handleRESTRequest(c *ctx.Context) {
	var catchEvent string

	defer func() {
		if rec := recover(); rec != nil {

			// Pipe errors run first
			// then exception filter
			if errorAggregationOperators, ok := c.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)).([]aggregation.AggregationOperator); ok {
				totalErrorAggregations := len(errorAggregationOperators)

				// Handle case if pipe error panic
				defer func() {
					if rec := recover(); rec != nil {
						c.Event.Emit(catchEvent, c, rec, 0)
					}
				}()

				for i := totalErrorAggregations - 1; i >= 0; i-- {
					aggregation := errorAggregationOperators[i]
					rec = aggregation(c, rec)
				}
			}

			// Execute exception filters if any
			// normally this one always ok
			// since we always set global exception filter as default
			if _, ok := http.catchRESTFnsMap[catchEvent]; ok && rec != nil {

				// 3rd param is index of catch function
				c.Event.Emit(catchEvent, c, rec, 0)
			}
		}
	}()

	isNext := true
	c.Next = func() {
		isNext = true
	}

	version := ""
	if http.isEnableVersioning {
		version = http.versioning.GetVersion(c)
	}

	isMatched, matchedRoute, paramKeys, paramValues, handlers := http.route.Match(c.Method, c.URL.Path, version)
	if !isMatched {
		isMatched, matchedRoute, paramKeys, paramValues, handlers = http.route.Match(c.Method, c.URL.Path, versioning.NEUTRAL_VERSION)
	}

	if http.isEnableVersioning {
		if version == "" && isMatched {
			// Invoke middlewares
			for _, middleware := range http.route.GlobalMiddlewares {
				if isNext {
					isNext = false
					middleware(c)
				}
			}

			if isNext {
				http.returnDeprecatedURL(c)
			}

			return
		}
	}

	catchEvent = matchedRoute

	if isMatched {
		c.SetRoute(matchedRoute)
		c.ParamKeys = paramKeys
		c.ParamValues = paramValues
		if c.Method == stdHTTP.MethodPost {
			c.Status(stdHTTP.StatusCreated)
		}

		for _, handler := range handlers {
			if isNext {
				isNext = false
				if handler == nil {

					// handler = nil / main handler
					// meaning this is injectable handler
					injectableHandler := http.route.InjectableHandlers[matchedRoute]

					// data return from main handler
					data := http.invokeHandler(injectableHandler, c)

					if aggregations, ok := c.Context().Value(WithValueKey(matchedRoute)).([]*aggregation.Aggregation); ok {
						var aggregatedData any
						isMainHandlerCalled := true

						totalAggregations := len(aggregations)

						for i := totalAggregations - 1; i >= 0; i-- {
							aggregation := aggregations[i]

							if aggregation.IsMainHandlerCalled {

								// set data from main handler into
								// first interceptor
								if i == totalAggregations-1 {
									if len(data) == 1 {
										aggregatedData = data[0].Interface()
									} else if len(data) > 1 {
										setStatusCode(c, data[0])
										aggregatedData = data[1].Interface()
									}
								}

								aggregation.SetMainData(aggregatedData)
								aggregatedData = aggregation.Aggregate(c)
							} else {
								isMainHandlerCalled = false
								if lastWildcardSlashIndex, ok := http.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
									var dir any

									if len(data) == 1 {
										dir = data[0].Interface()
									} else if len(data) > 1 {
										setStatusCode(c, data[0])
										dir = data[1].Interface()
									}
									http.serveContent(c, lastWildcardSlashIndex, dir)
								} else {
									returnREST(c, reflect.ValueOf(aggregation.InterceptorData))
								}
								break
							}
						}

						if isMainHandlerCalled {
							if lastWildcardSlashIndex, ok := http.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
								var dir any

								if len(data) == 1 {
									dir = data[0].Interface()
								} else if len(data) > 1 {
									setStatusCode(c, data[0])
									dir = data[1].Interface()
								}
								http.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnREST(c, reflect.ValueOf(aggregatedData))
							}
						}
					} else {
						if len(data) == 1 {
							if lastWildcardSlashIndex, ok := http.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
								dir := data[0].Interface()
								http.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnREST(c, data[0])
							}
						} else if len(data) > 1 {
							setStatusCode(c, data[0])
							if lastWildcardSlashIndex, ok := http.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
								dir := data[1].Interface()
								http.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnREST(c, data[1])
							}
						}
					}
				} else {
					handler(c)
				}
			}
		}
	} else {
		// Invoke middlewares
		for _, middleware := range http.route.GlobalMiddlewares {
			if isNext {
				isNext = false
				middleware(c)
			}
		}

		if isNext {
			http.returnNotFound(c)
		}
	}
}

func (http *HTTP) getContextID(c *ctx.Context) string {
	reqID := c.Header().Get(ctx.REQUEST_ID)
	if reqID == "" {
		uuid, _ := utils.StrUUID()
		return uuid
	}

	return reqID
}

func (http *HTTP) serveContent(c *ctx.Context, lastWildcardSlashIndex int, dir any) {
	if dir, ok := dir.(string); ok {
		if lastWildcardSlashIndex != 0 {
			urlPath := utils.StrRemoveDup(c.URL.Path, "/")
			urlPathArr := strings.Split(urlPath, "/")
			suffix := strings.Join(urlPathArr[lastWildcardSlashIndex:], "/")
			oldDir := dir
			dir = path.Join(dir, suffix)

			if len(dir) < len(oldDir) {
				http.returnInvalidURL(c)
				return
			}
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) || err != nil {
			http.returnNotFound(c)
		} else {
			stdHTTP.ServeFile(c.ResponseWriter, c.Request, dir)
			c.Event.Emit(ctx.REQUEST_FINISHED, c)
		}
	} else {
		http.returnNotFound(c)
	}
}

func (http *HTTP) returnNotFound(c *ctx.Context) {
	notFoundException := exception.NotFoundException("Cannot " + c.Method + " " + c.URL.Path)
	httpCode, _ := notFoundException.GetHTTPStatus()
	c.Status(httpCode)
	c.JSON(ctx.Map{
		"code":    notFoundException.GetCode(),
		"error":   notFoundException.Error(),
		"message": notFoundException.GetResponse(),
	})
}

func (http *HTTP) returnInvalidURL(c *ctx.Context) {
	badRequestException := exception.BadRequestException("Invalid URL path")
	httpCode, _ := badRequestException.GetHTTPStatus()
	c.Status(httpCode)
	c.JSON(ctx.Map{
		"code":    badRequestException.GetCode(),
		"error":   badRequestException.Error(),
		"message": badRequestException.GetResponse(),
	})
}

func (http *HTTP) returnDeprecatedURL(c *ctx.Context) {
	goneException := exception.GoneException("Deprecated URL usage")
	httpCode, _ := goneException.GetHTTPStatus()
	c.Status(httpCode)
	c.JSON(ctx.Map{
		"code":    goneException.GetCode(),
		"error":   goneException.Error(),
		"message": goneException.GetResponse(),
	})
}
