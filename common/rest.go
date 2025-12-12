package common

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/utils"
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
	TOKEN_BY      = "BY"
	TOKEN_AND     = "AND"
	TOKEN_OF      = "OF"
	TOKEN_ANY     = "ANY"
	TOKEN_FILE    = "FILE"
	TOKEN_VERSION = "VERSION"
)

var TokenMap = map[string]string{
	TOKEN_BY:      TOKEN_BY,
	TOKEN_AND:     TOKEN_AND,
	TOKEN_OF:      TOKEN_OF,
	TOKEN_ANY:     TOKEN_ANY,
	TOKEN_FILE:    TOKEN_FILE,
	TOKEN_VERSION: TOKEN_VERSION,
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
	prefixes           []Prefix
	PatternToFnNameMap map[string]string
	FnNameToPatternMap map[string]string
	RouterMap          map[string]any
}

type Prefix struct {
	Value    string
	Handlers []any
}

func (r *REST) addToRouters(fnName, route, version, method string, injectableHandler any) {
	if reflect.ValueOf(r.RouterMap).IsNil() {
		r.RouterMap = make(map[string]any)
	}

	if r.PatternToFnNameMap == nil {
		r.PatternToFnNameMap = map[string]string{}
	}

	if r.FnNameToPatternMap == nil {
		r.FnNameToPatternMap = map[string]string{}
	}

	pattern := routing.MethodRouteVersionToPattern(method, route, version)

	r.RouterMap[pattern] = injectableHandler
	r.PatternToFnNameMap[pattern] = fnName
	r.FnNameToPatternMap[fnName] = pattern
}

func (r *REST) GetPrefixes() []map[string]string {
	prefixes := []map[string]string{}

	for _, prefixConf := range r.prefixes {
		prefixValue := routing.ToEndpoint(prefixConf.Value)
		prefixHandlers := prefixConf.Handlers

		// if no handlers were binded
		// then prefix will be applied for all handlers
		if len(prefixHandlers) == 0 {
			prefixMap := make(map[string]string)
			prefixMap[prefixValue] = "*"
			prefixes = append(prefixes, prefixMap)
		} else {
			for _, handler := range prefixHandlers {
				prefixMap := make(map[string]string)
				prefixMap[prefixValue] = GetFnName(handler)
				prefixes = append(prefixes, prefixMap)
			}
		}
	}

	return prefixes
}

func (r *REST) addPrefixesToRoute(route, fnName string, prefixes []map[string]string) string {
	for _, prefix := range prefixes {
		for prefixValue, prefixFnName := range prefix {
			if prefixFnName == "*" || prefixFnName == fnName {
				route = prefixValue + route
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

	httpMethod, route, version := ParseFnNameToURL(fnName, RESTOperations)
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
				utils.FmtRed(
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
	routes := []RESTConfiguration{}

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
