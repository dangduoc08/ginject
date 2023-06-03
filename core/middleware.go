package core

import (
	"fmt"
	"reflect"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type MiddlewareConfig struct {
	middleware *Middleware
}

type Middleware struct {
	middlewares       []context.Handler
	inclusion         []string
	exclusion         []string
	middlewareItemArr []struct {
		method   string
		route    string
		handlers []context.Handler
	}
}

func (mw *Middleware) Apply(middlewares ...context.Handler) *MiddlewareConfig {
	if len(mw.middlewares) > 0 {
		mw.add([]map[string]string{})
	}

	mw.middlewares = middlewares

	// set exclusion empty
	// to prevent only invoke Use not invoke Exclude
	mw.exclusion = []string{}
	return &MiddlewareConfig{
		middleware: mw,
	}
}

func (mw *Middleware) include(methodName string) *Middleware {
	httpMethod, _ := common.ParseFnNameToURL(methodName)
	if httpMethod != "" {
		mw.inclusion = append(mw.inclusion, methodName)
	}

	return mw
}

func (mw *Middleware) add(prefixes []map[string]string) {
	for _, fnNameInclusion := range mw.inclusion {

		if !utils.ArrIncludes[string](mw.exclusion, fnNameInclusion) {
			httpMethod, route := common.ParseFnNameToURL(fnNameInclusion)
			if httpMethod != "" {
				for _, prefix := range prefixes {
					for prefixValue, prefixFnName := range prefix {
						if prefixFnName == "ALL" || prefixFnName == fnNameInclusion {
							route = prefixValue + route
						}
					}
				}

				// apply for all
				mw.middlewareItemArr = append(mw.middlewareItemArr, struct {
					method   string
					route    string
					handlers []func(*context.Context)
				}{
					method:   httpMethod,
					route:    routing.ToEndpoint(route),
					handlers: mw.middlewares,
				})
			}
		}
	}
}

func (mc *MiddlewareConfig) Exclude(configExclusion []any) *Middleware {
	for _, handler := range configExclusion {
		handlerKind := reflect.TypeOf(handler).Kind()
		if handler == nil || handlerKind != reflect.Func {
			panic(fmt.Errorf(
				utils.FmtRed(
					"%v is not a handler",
					handlerKind,
				),
			))
		}

		mc.middleware.exclusion = append(mc.middleware.exclusion, common.GetFnName(handler))
	}

	return mc.middleware
}
