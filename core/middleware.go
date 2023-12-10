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
	middlewares           []context.Handler
	inclusion             []string
	exclusion             []string
	restMiddlewareItemArr []struct {
		method   string
		route    string
		handlers []context.Handler
	}
	wsMiddlewareItemArr []struct {
		eventName string
		handlers  []context.Handler
	}
}

func (mw *Middleware) Apply(middlewares ...context.Handler) *MiddlewareConfig {

	// when Apply invoked twice on same module
	if len(mw.middlewares) > 0 {
		mw.addREST([]map[string]string{})
		mw.addWS()
	}

	mw.middlewares = middlewares

	// set exclusion empty
	// to prevent only invoke Use not invoke Exclude
	mw.exclusion = []string{}
	return &MiddlewareConfig{
		middleware: mw,
	}
}

func (mw *Middleware) includeREST(methodName string) *Middleware {
	httpMethod, _ := common.ParseFnNameToURL(methodName, common.RESTOperations)
	if httpMethod != "" {
		mw.inclusion = append(mw.inclusion, methodName)
	}

	return mw
}

func (mw *Middleware) addREST(prefixes []map[string]string) {
	for _, fnNameInclusion := range mw.inclusion {

		if !utils.ArrIncludes[string](mw.exclusion, fnNameInclusion) {
			httpMethod, route := common.ParseFnNameToURL(fnNameInclusion, common.RESTOperations)
			if httpMethod != "" {
				for _, prefix := range prefixes {
					for prefixValue, prefixFnName := range prefix {
						if prefixFnName == "*" || prefixFnName == fnNameInclusion {
							route = prefixValue + route
						}
					}
				}

				// apply for all
				mw.restMiddlewareItemArr = append(mw.restMiddlewareItemArr, struct {
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

func (mw *Middleware) addWS() {
	for _, patternFNNameInclusion := range mw.inclusion {

		// because line 113
		// have to resolve to get fnName and subprotocol
		opr, eventname := common.ParseFnNameToURL(patternFNNameInclusion, common.WSOperations)
		if opr != "" {
			eventname = utils.StrRemoveEnd(utils.StrRemoveBegin(eventname, "/"), "/")
			_, fnNameInclusion := context.ResolveWSEventname(eventname)
			if !utils.ArrIncludes[string](mw.exclusion, fnNameInclusion) {
				mw.wsMiddlewareItemArr = append(mw.wsMiddlewareItemArr, struct {
					eventName string
					handlers  []func(*context.Context)
				}{
					eventName: eventname,
					handlers:  mw.middlewares,
				})
			}
		}
	}
}

func (mw *Middleware) includeWS(subprotocol, methodName string) *Middleware {
	opr, event := common.ParseFnNameToURL(methodName, common.WSOperations)
	if opr != "" {
		mw.inclusion = append(mw.inclusion, opr+"_"+common.ToWSEventName(subprotocol, event))
	}

	return mw
}

func (mc *MiddlewareConfig) Exclude(configExclusion ...any) *Middleware {
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
