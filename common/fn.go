package common

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

// to ensure constructor only run once
var singletons = make(map[string]any)

func GetFuncName(handler any) string {
	name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return strings.TrimSuffix(name, "-fm")
}

func ParseFuncNameToURL(fnName string) (string, string, string) {
	method := ""
	route := ""
	version := ""

	subStr := strings.FieldsFunc(fnName, func(r rune) bool { return r == '_' })
	j := -1

	for i, b := range subStr {

		// when set j = i
		// mean it's skip
		if j >= 0 && i < j {
			continue
		}

		s := b

		if i == 0 {
			var ok bool
			method, ok = RESTOperations[s]
			if !ok {
				return "", "", ""
			}
		}

		if s == TokenVersion {
			if i+1 < len(subStr) {
				version = strings.Join(subStr[i+1:], "_")
			}
			break
		}

		if _, ok := RESTOperations[s]; ok || s == TokenOf {
			i++
			path := ""
			isAny := false

			for i < len(subStr) &&
				subStr[i] != TokenBy &&
				subStr[i] != TokenAnd &&
				subStr[i] != TokenOf &&
				subStr[i] != TokenVersion {

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

		if s == TokenBy || s == TokenAnd {
			firstSlashIndex := strings.Index(route, "/")
			shouldConcatRoute := route[:firstSlashIndex]
			remainRoutes := route[firstSlashIndex:]

			i++
			start := i
			for i < len(subStr) && RESTTokenMap[subStr[i]] == "" {
				i++
			}
			param := strings.Join(subStr[start:i], "_")
			j = i

			if firstSlashIndex > -1 && firstSlashIndex < len(route)-1 {
				if route[firstSlashIndex+1:firstSlashIndex+2] == "{" {
					firstParamIndex := strings.Index(remainRoutes, "}/")
					if firstParamIndex > -1 {
						route = shouldConcatRoute + remainRoutes[:firstParamIndex+1] + "/{" + param + "}" + remainRoutes[firstParamIndex+1:]
					}
				} else {
					route = shouldConcatRoute + "/{" + param + "}" + remainRoutes
				}
			} else {
				route = shouldConcatRoute + "/{" + param + "}" + remainRoutes
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
			route = "*/" + shouldConcatRoute + remainRoutes
			continue
		}
	}

	return method, "/" + strings.TrimPrefix(route, "/"), version
}

func ParseWSFuncNameToEvent(fnName string) (string, bool) {
	op, rest, found := strings.Cut(fnName, "_")
	if !found || rest == "" {
		return "", false
	}
	if _, ok := WSOperations[op]; !ok {
		return "", false
	}
	parts := strings.Split(rest, "_")
	segs := parts[:0]
	for _, p := range parts {
		switch p {
		case "":
		case TokenAny:
			segs = append(segs, "*")
		default:
			segs = append(segs, strings.ToLower(p))
		}
	}
	if len(segs) == 0 {
		return "", false
	}
	return strings.Join(segs, "."), true
}

func HandleGuard(c *ctx.Context, canActive bool) {
	if canActive {
		c.Next()
	} else {
		forbiddenException := exception.ForbiddenException("Access denied")
		panic(forbiddenException)
	}
}

func Construct(obj any, constructor string) any {
	newObjValue := reflect.ValueOf(obj)
	if newObj, ok := singletons[newObjValue.String()]; ok {
		return newObj
	}

	objConstructor := newObjValue.MethodByName(constructor)
	if objConstructor.IsValid() {
		obj = objConstructor.Call([]reflect.Value{})[0].Interface()
		singletons[newObjValue.String()] = obj
	}

	return obj
}

func ToWSEventName(s string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, "/"), "/")
}
