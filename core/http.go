package core

import (
	stdHTTP "net/http"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/versioning"
)

type HTTP struct {
	route *routing.Router

	versioning                    *versioning.Versioning
	isVersioningEnabled           bool
	catchFnsByRoute               map[string][]common.HTTPCatch
	lastWildcardSlashIndexByRoute map[string]int

	resolveAndCallHandler func(f any, c *ctx.HTTPContext) []reflect.Value
}

func newHTTP() *HTTP {
	return &HTTP{
		route:                         routing.NewRouter(),
		catchFnsByRoute:               make(map[string][]common.HTTPCatch),
		lastWildcardSlashIndexByRoute: make(map[string]int),
	}
}

func (http *HTTP) enableVersioning(v versioning.Versioning) {
	if v.Key == "" {
		v.Key = "v"
	}
	http.versioning = &v
	http.isVersioningEnabled = true
}

func (http *HTTP) addMainHandler(moduleHandler common.HTTPLayer) {
	httpMethod := routing.OperationsMapHTTPMethods[moduleHandler.Method]
	if moduleHandler.Method == routing.SERVE {
		r := moduleHandler.Route
		lr := len(r)
		lastWildcardSlashIndex := 0 // zero mean use config dir
		if lr >= 2 && r[lr-2:] == "*/" {
			lastWildcardSlashIndex = strings.Count(r, "/") - 1
		}

		http.lastWildcardSlashIndexByRoute[routing.MethodRouteVersionToPattern(httpMethod, moduleHandler.Route, moduleHandler.Version)] = lastWildcardSlashIndex
	}
	http.route.AddInjectableHandler(httpMethod, moduleHandler.Route, moduleHandler.Version, moduleHandler.Handler)
}

func (http *HTTP) handleRequest(c *ctx.HTTPContext) {
	var catchEvent string

	defer func() {
		if rec := recover(); rec != nil {
			if catchFns, ok := http.catchFnsByRoute[catchEvent]; ok {
				common.RunHTTPCatchChain(c, catchFns, rec)
			}
		}
	}()

	isNext := true
	c.Next = func() {
		isNext = true
	}

	version := ""
	if http.isVersioningEnabled {
		version = http.versioning.GetVersion(c)
	}

	isMatched, matchedRoute, paramKeys, paramValues, handlers := http.route.Match(c.Method, c.URL.Path, version)
	if !isMatched {
		isMatched, matchedRoute, paramKeys, paramValues, handlers = http.route.Match(c.Method, c.URL.Path, versioning.NeutralVersion)
	}

	if http.isVersioningEnabled {
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
					data := http.resolveAndCallHandler(injectableHandler, c)

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
								aggregatedData = aggregation.Aggregate()
							} else {
								isMainHandlerCalled = false
								if lastWildcardSlashIndex, ok := http.lastWildcardSlashIndexByRoute[matchedRoute]; ok {
									var dir any

									if len(data) == 1 {
										dir = data[0].Interface()
									} else if len(data) > 1 {
										setStatusCode(c, data[0])
										dir = data[1].Interface()
									}
									http.serveContent(c, lastWildcardSlashIndex, dir)
								} else {
									returnHTTP(c, reflect.ValueOf(aggregation.InterceptorData))
								}
								break
							}
						}

						if isMainHandlerCalled {
							if lastWildcardSlashIndex, ok := http.lastWildcardSlashIndexByRoute[matchedRoute]; ok {
								var dir any

								if len(data) == 1 {
									dir = data[0].Interface()
								} else if len(data) > 1 {
									setStatusCode(c, data[0])
									dir = data[1].Interface()
								}
								http.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnHTTP(c, reflect.ValueOf(aggregatedData))
							}
						}
					} else {
						if len(data) == 1 {
							if lastWildcardSlashIndex, ok := http.lastWildcardSlashIndexByRoute[matchedRoute]; ok {
								dir := data[0].Interface()
								http.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnHTTP(c, data[0])
							}
						} else if len(data) > 1 {
							setStatusCode(c, data[0])
							if lastWildcardSlashIndex, ok := http.lastWildcardSlashIndexByRoute[matchedRoute]; ok {
								dir := data[1].Interface()
								http.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnHTTP(c, data[1])
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

func (http *HTTP) serveContent(c *ctx.HTTPContext, lastWildcardSlashIndex int, dir any) {
	if dir, ok := dir.(string); ok {
		if lastWildcardSlashIndex != 0 {
			urlPath := str.RemoveDup(c.URL.Path, "/")
			urlPathArr := strings.Split(urlPath, "/")
			suffix := strings.Join(urlPathArr[lastWildcardSlashIndex:], "/")
			oldDir := dir
			dir = path.Join(dir, suffix)

			if dir != oldDir && !strings.HasPrefix(dir, oldDir+"/") {
				http.returnInvalidURL(c)
				return
			}
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) || err != nil {
			http.returnNotFound(c)
		} else {
			stdHTTP.ServeFile(c.ResponseWriter, c.Request, dir)
			c.Event.Emit(ctx.RequestFinished, c)
		}
	} else {
		http.returnNotFound(c)
	}
}

func (http *HTTP) returnNotFound(c *ctx.HTTPContext) {
	notFoundException := exception.NotFoundException("Cannot " + c.Method + " " + c.URL.Path)
	c.Status(notFoundException.GetCode())
	c.JSON(ctx.Map{
		"code":    notFoundException.GetCode(),
		"error":   notFoundException.Error(),
		"message": notFoundException.GetMessage(),
	})
}

func (http *HTTP) returnInvalidURL(c *ctx.HTTPContext) {
	badRequestException := exception.BadRequestException("Invalid URL path")
	c.Status(badRequestException.GetCode())
	c.JSON(ctx.Map{
		"code":    badRequestException.GetCode(),
		"error":   badRequestException.Error(),
		"message": badRequestException.GetMessage(),
	})
}

func (http *HTTP) returnDeprecatedURL(c *ctx.HTTPContext) {
	goneException := exception.GoneException("Deprecated URL usage")
	c.Status(goneException.GetCode())
	c.JSON(ctx.Map{
		"code":    goneException.GetCode(),
		"error":   goneException.Error(),
		"message": goneException.GetMessage(),
	})
}
