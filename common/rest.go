package common

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/routing"
)

var RESTOperations = map[string]string{
	"READ":        http.MethodGet,
	"CREATE":      http.MethodPost,
	"UPDATE":      http.MethodPut,
	"MODIFY":      http.MethodPatch,
	"DELETE":      http.MethodDelete,
	"PREFLIGHT":   http.MethodOptions,
	routing.SERVE: routing.SERVE,
}

var InsertedRoutes = make(map[string]string)

const (
	TokenBy      = "BY"
	TokenAnd     = "AND"
	TokenOf      = "OF"
	TokenAny     = "ANY"
	TokenAll     = "ALL"
	TokenFile    = "FILE"
	TokenVersion = "VERSION"
)

var RESTTokenMap = map[string]string{
	TokenBy:      TokenBy,
	TokenAnd:     TokenAnd,
	TokenOf:      TokenOf,
	TokenAny:     TokenAny,
	TokenFile:    TokenFile,
	TokenVersion: TokenVersion,
}

type RESTLayer struct {
	Handler         any
	ControllerPath  string
	Name            string
	Route           string
	Version         string
	Method          string
	Pattern         string
	MainHandlerName string
}

type RESTConfiguration struct {
	Method  string
	Route   string
	Version string
	Func    string
}

type REST struct {
	prefixes             []Prefix
	PatternToFuncNameMap map[string]string
	FuncNameToPatternMap map[string]string
	RouterMap            map[string]any
}

type Prefix struct {
	Value    string
	Handlers []any
}

func (r *REST) addToRouters(fnName, route, version, method string, injectableHandler any) {
	if r.RouterMap == nil {
		r.RouterMap = make(map[string]any)
	}

	if r.PatternToFuncNameMap == nil {
		r.PatternToFuncNameMap = map[string]string{}
	}

	if r.FuncNameToPatternMap == nil {
		r.FuncNameToPatternMap = map[string]string{}
	}

	pattern := routing.MethodRouteVersionToPattern(method, route, version)

	r.RouterMap[pattern] = injectableHandler
	r.PatternToFuncNameMap[pattern] = fnName
	r.FuncNameToPatternMap[fnName] = pattern
}

func (r *REST) GetPrefixes() []map[string]string {
	prefixes := make([]map[string]string, 0, len(r.prefixes))

	for _, prefixConf := range r.prefixes {
		prefixValue := routing.ToEndpoint(prefixConf.Value)
		prefixHandlers := prefixConf.Handlers

		// if no handlers were binded
		// then prefix will be applied for all handlers
		if len(prefixHandlers) == 0 {
			prefixes = append(prefixes, map[string]string{prefixValue: "*"})
		} else {
			for _, handler := range prefixHandlers {
				prefixes = append(prefixes, map[string]string{prefixValue: GetFuncName(handler)})
			}
		}
	}

	return prefixes
}

func (r *REST) addPrefixesToRoute(route, fnName string, prefixes []map[string]string) string {
	for _, prefix := range prefixes {
		for prefixValue, prefixFnName := range prefix {
			if prefixFnName == "*" || prefixFnName == fnName {
				route = prefixValue + strings.TrimPrefix(route, "/")
			}
		}
	}

	return route
}

func (r *REST) Prefix(v string, handlers ...any) *REST {
	r.prefixes = append([]Prefix{
		{
			Value:    v,
			Handlers: handlers,
		},
	}, r.prefixes...)

	return r
}

func (r *REST) AddHandlerToRouterMap(modulePrefixes []string, fnName string, handler any) {
	prefixes := r.GetPrefixes()

	httpMethod, route, version := ParseFuncNameToURL(fnName)
	if httpMethod != "" {
		route = r.addPrefixesToRoute(route, fnName, prefixes)
		for _, modulePrefix := range modulePrefixes {
			route = modulePrefix + route
		}

		pattern := routing.MethodRouteVersionToPattern(httpMethod, route, version)
		if InsertedRoutes[pattern] == "" {
			InsertedRoutes[pattern] = fnName
		} else {
			panic(errors.New(
				color.FmtRed(
					"%v method is conflicted with %v method",
					fnName,
					InsertedRoutes[pattern],
				),
			))
		}

		r.addToRouters(fnName, route, version, httpMethod, handler)
	}
}

func (r *REST) GetConfigurations() []RESTConfiguration {
	routes := make([]RESTConfiguration, 0, len(InsertedRoutes))

	for routeMethod, fn := range InsertedRoutes {
		method, route, version := routing.PatternToMethodRouteVersion(routeMethod)
		routes = append(routes, RESTConfiguration{
			Method:  method,
			Route:   route,
			Version: version,
			Func:    fn,
		})
	}

	return routes
}
