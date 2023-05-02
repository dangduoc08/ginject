package common

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

var Operations = map[string]string{
	"READ":   http.MethodGet,
	"CREATE": http.MethodPost,
	"UPDATE": http.MethodPut,
	"MODIFY": http.MethodPatch,
	"DELETE": http.MethodDelete,
	"DO":     "DO",
}

const (
	TokenBy   = "BY"
	TokenAnd  = "AND"
	TokenOf   = "OF"
	TokenAny  = "ANY"
	TokenFile = "FILE"
)

var TokenMap = map[string]string{
	TokenBy:   TokenBy,
	TokenAnd:  TokenAnd,
	TokenOf:   TokenOf,
	TokenAny:  TokenAny,
	TokenFile: TokenFile,
}

type Rest struct {
	prefixes  []Prefix
	RouterMap map[string]any
}

type Prefix struct {
	Value    string
	Handlers []any
}

func (r *Rest) addToRouters(path, method string, injectableHandler any) {
	if reflect.ValueOf(r.RouterMap).IsNil() {
		r.RouterMap = make(map[string]any)
	}
	r.RouterMap[routing.AddMethodToRoute(routing.ToEndpoint(path), method)] = injectableHandler
}

func (r *Rest) addAllToRouters(path string, injectableHandler any) {
	for _, method := range Operations {
		if method != Operations["DO"] {
			r.addToRouters(path, method, injectableHandler)
		}
	}
}

func (r *Rest) Prefix(v string, handlers ...any) *Rest {
	r.prefixes = append([]Prefix{
		{
			Value:    v,
			Handlers: handlers,
		},
	}, r.prefixes...)

	return r
}

func (r *Rest) GenPrefixes() []map[string]string {
	prefixes := []map[string]string{}

	for _, prefixConf := range r.prefixes {
		prefixMap := make(map[string]string)
		prefixValue := utils.StrAddBegin(utils.StrRemoveEnd(utils.StrRemoveSpace(prefixConf.Value), "/"), "/")
		prefixHandlers := prefixConf.Handlers
		if len(prefixHandlers) == 0 {
			prefixMap[prefixValue] = "all"
		} else {
			for _, handler := range prefixHandlers {
				prefixMap[prefixValue] = getFnName(handler)
			}
		}

		prefixes = append(prefixes, prefixMap)
	}

	return prefixes
}

func (r *Rest) AddHandlerToRouterMap(fnName string, insertedRoutes map[string]string, prefixes []map[string]string, handler any) {
	httpMethod, route := r.ParseFnNameToURL(fnName)
	if httpMethod != "" {
		for _, prefix := range prefixes {
			for prefixValue, prefixFnName := range prefix {
				if prefixFnName == "all" || prefixFnName == fnName {
					route = prefixValue + route
				}
			}
		}

		parsedRoute, _ := routing.ParseToParamKey(routing.AddMethodToRoute(routing.ToEndpoint(route), httpMethod))
		if insertedRoutes[parsedRoute] == "" {
			insertedRoutes[parsedRoute] = fnName
		} else {
			panic(fmt.Errorf(
				utils.FmtRed(
					"%v method is conflicted with %v method",
					fnName,
					insertedRoutes[parsedRoute],
				),
			))
		}

		if httpMethod == Operations["DO"] {
			r.addAllToRouters(route, handler)
		} else {
			r.addToRouters(route, httpMethod, handler)
		}
	}
}

func (r *Rest) ParseFnNameToURL(fnName string) (string, string) {
	return r.segmentFnName(fnName)
}

func (r *Rest) segmentFnName(fnName string) (string, string) {
	method := ""
	route := ""

	subStr := strings.Split(fnName, "_")
	j := -1

	for i, b := range subStr {
		if j >= 0 && i < j {
			continue
		}

		s := string(b)

		// function name is not satisfied statements
		if _, ok := Operations[s]; !ok && i == 0 {
			return "", ""
		}

		if _, ok := Operations[s]; ok && i == 0 {
			method = Operations[s]
		}

		if _, ok := Operations[s]; ok || s == TokenOf {
			i++
			path := ""
			isAny := false

			for i < len(subStr) &&
				subStr[i] != TokenBy &&
				subStr[i] != TokenAnd &&
				subStr[i] != TokenOf {

				// READ_ANY
				// or OF_ANY
				// mapped with condition line 54
				if subStr[i] == TokenAny {
					path += "*"
					isAny = true
				}

				if subStr[i] == TokenFile {
					lastWildcardIndex := strings.LastIndex(path, "*")
					if lastWildcardIndex > -1 {
						remainPath := "*"
						extension := strings.ToLower(path[lastWildcardIndex+1:])
						path = remainPath + "." + extension
					} else {
						lastWildcardIndex := strings.LastIndex(path, "_")
						if lastWildcardIndex > -1 {
							remainPath := path[:lastWildcardIndex]
							if remainPath == TokenAny {
								remainPath = "*"
							}
							extension := strings.ToLower(path[lastWildcardIndex+1:])

							path = remainPath + "." + extension
						}
					}
				}

				if subStr[i] != TokenAny && subStr[i] != TokenFile {
					if path == "" || isAny {
						path += subStr[i]
						isAny = false
					} else {
						path += "_" + subStr[i]
					}
				}
				i++
			}
			j = i

			route = path + "/" + route
			continue
		}

		// param concat to first slash of path
		if s == TokenBy || s == TokenAnd {
			firstSlashIndex := strings.Index(route, "/")
			shouldConcatRoute := route[:firstSlashIndex]
			remainRoutes := route[firstSlashIndex:]

			param := ""
			i++
			for i < len(subStr) && TokenMap[subStr[i]] == "" {
				if param == "" {
					param += subStr[i]
				} else {
					param += "_" + subStr[i]
				}
				i++
			}
			j = i

			if firstSlashIndex > -1 && firstSlashIndex < len(route)-1 {
				if route[firstSlashIndex+1:firstSlashIndex+2] == "{" {
					firstParamIndex := strings.Index(remainRoutes, "}/")
					if firstParamIndex > -1 {
						route = fmt.Sprintf("%v%v/{%v}%v", shouldConcatRoute, remainRoutes[:firstParamIndex+1], param, remainRoutes[firstParamIndex+1:])
					}
				} else {
					route = fmt.Sprintf("%v/{%v}%v", shouldConcatRoute, param, remainRoutes)
				}
			} else {
				route = fmt.Sprintf("%v/{%v}%v", shouldConcatRoute, param, remainRoutes)
			}
			continue
		}

		// ANY stand alone
		if s == TokenAny && (i == len(subStr)-1 || subStr[i+1] == TokenOf) {

			// ANY same as a static path
			if route == "" {
				route = "*/"
				continue
			}
			firstSlashIndex := strings.Index(route, "/")
			shouldConcatRoute := route[:firstSlashIndex]
			remainRoutes := route[firstSlashIndex:]
			route = fmt.Sprintf("%v/%v%v", "*", shouldConcatRoute, remainRoutes)
			continue
		}
	}

	return method, "/" + route
}
