package common

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

var RESTOperations = map[string]string{
	"READ":   http.MethodGet,
	"CREATE": http.MethodPost,
	"UPDATE": http.MethodPut,
	"MODIFY": http.MethodPatch,
	"DELETE": http.MethodDelete,
}

var InsertedRoutes = make(map[string]string)

const (
	TOKEN_BY   = "BY"
	TOKEN_AND  = "AND"
	TOKEN_OF   = "OF"
	TOKEN_ANY  = "ANY"
	TOKEN_FILE = "FILE"
)

var TokenMap = map[string]string{
	TOKEN_BY:   TOKEN_BY,
	TOKEN_AND:  TOKEN_AND,
	TOKEN_OF:   TOKEN_OF,
	TOKEN_ANY:  TOKEN_ANY,
	TOKEN_FILE: TOKEN_FILE,
}

type RESTConfiguration struct {
	Method string
	Route  string
	Func   string
}

type REST struct {
	prefixes           []Prefix
	PatternToFnNameMap map[string]string
	RouterMap          map[string]any
}

type Prefix struct {
	Value    string
	Handlers []any
}

func (r *REST) addToRouters(fnName, path, method string, injectableHandler any) {
	if reflect.ValueOf(r.RouterMap).IsNil() {
		r.RouterMap = make(map[string]any)
	}

	if r.PatternToFnNameMap == nil {
		r.PatternToFnNameMap = map[string]string{}
	}
	pattern := routing.AddMethodToRoute(routing.ToEndpoint(path), method)

	r.RouterMap[pattern] = injectableHandler
	r.PatternToFnNameMap[pattern] = fnName
}

func (r *REST) GetPrefixes() []map[string]string {
	prefixes := []map[string]string{}

	for _, prefixConf := range r.prefixes {
		prefixMap := make(map[string]string)
		prefixValue := utils.StrAddBegin(utils.StrRemoveEnd(utils.StrRemoveSpace(prefixConf.Value), "/"), "/")
		prefixHandlers := prefixConf.Handlers

		// if no handlers were binded
		// then prefix will be applied for all handlers
		if len(prefixHandlers) == 0 {
			prefixMap[prefixValue] = "*"
		} else {
			for _, handler := range prefixHandlers {
				prefixMap[prefixValue] = GetFnName(handler)
			}
		}

		prefixes = append(prefixes, prefixMap)
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

func (r *REST) AddHandlerToRouterMap(fnName string, handler any) {
	prefixes := r.GetPrefixes()

	httpMethod, route := ParseFnNameToURL(fnName, RESTOperations)
	if httpMethod != "" {
		route = r.addPrefixesToRoute(route, fnName, prefixes)
		routeMethod := routing.AddMethodToRoute(routing.ToEndpoint(route), httpMethod)
		if InsertedRoutes[routeMethod] == "" {
			InsertedRoutes[routeMethod] = fnName
		} else {
			panic(fmt.Errorf(
				utils.FmtRed(
					"%v method is conflicted with %v method",
					fnName,
					InsertedRoutes[routeMethod],
				),
			))
		}

		r.addToRouters(fnName, route, httpMethod, handler)
	}
}

func (r *REST) GetConfigurations() []RESTConfiguration {
	routes := []RESTConfiguration{}

	for routeMethod, fn := range InsertedRoutes {
		method, route := routing.SplitRoute(routeMethod)
		routes = append(routes, RESTConfiguration{
			Method: method,
			Route:  route,
			Func:   fn,
		})
	}

	return routes
}
