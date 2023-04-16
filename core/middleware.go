package core

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type RouteConfig struct {
	Path   string
	Method string
}

type MiddlewareConfig struct {
	middleware *Middleware
}

type Middleware struct {
	middlewares      []context.Handler
	inclusion        []RouteConfig
	exclusion        []RouteConfig
	middlewareMapArr []map[string]struct {
		handlers []context.Handler
		methods  []string
	}
}

func (mw *Middleware) Apply(middlewares ...context.Handler) *MiddlewareConfig {
	if len(mw.middlewares) > 0 {
		mw.add()
	}

	mw.middlewares = middlewares

	// set exclusion empty
	// to prevent only invoke Use not invoke Exclude
	mw.exclusion = []RouteConfig{}
	return &MiddlewareConfig{
		middleware: mw,
	}
}

func (mw *Middleware) include(configInclusion []RouteConfig) *Middleware {
	mw.inclusion = append(mw.inclusion, configInclusion...)
	return mw
}

func (mw *Middleware) add() {
	middlewareMap := make(map[string]struct {
		handlers []context.Handler
		methods  []string
	})

	for _, routeConfig := range mw.inclusion {
		methods := []string{}

		// method = ""
		// apply for all
		if routeConfig.Method == "" {
			methods = append(methods, routing.HTTPMethods...)
		} else {
			methods = append(methods, routeConfig.Method)
		}

		if configs, ok := middlewareMap[routeConfig.Path]; ok {
			configs.methods = append(configs.methods, methods...)
			configs.methods = utils.ArrToUnique(configs.methods)
			middlewareMap[routeConfig.Path] = configs
		} else {
			middlewareMap[routeConfig.Path] = struct {
				handlers []func(c *context.Context)
				methods  []string
			}{
				handlers: mw.middlewares,
				methods:  methods,
			}
		}
	}

	for _, exclusionConfig := range mw.exclusion {
		if configs, ok := middlewareMap[exclusionConfig.Path]; ok {

			// exclude for all
			if exclusionConfig.Method == "" {
				configs.methods = []string{}
			} else {
				matchedIndex := utils.ArrFindIndex(configs.methods, func(inclusionMethod string, i int) bool {
					return inclusionMethod == exclusionConfig.Method
				})
				configs.methods = utils.ArrFilter(configs.methods, func(el string, i int) bool {
					return i != matchedIndex
				})
			}

			middlewareMap[exclusionConfig.Path] = configs
		}
	}

	mw.middlewareMapArr = append(mw.middlewareMapArr, middlewareMap)
}

func (mc *MiddlewareConfig) Exclude(configExclusion []RouteConfig) *Middleware {
	mc.middleware.exclusion = configExclusion
	return mc.middleware
}
