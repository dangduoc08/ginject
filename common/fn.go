package common

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/utils"
)

// to ensure constructor only run once
var singletons = make(map[string]any)

func GetFnName(handler any) string {
	strs := strings.Split(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), ".")
	fnName := strs[len(strs)-1]
	fnName = strings.TrimSuffix(fnName, "-fm")
	return fnName
}

func ParseFnNameToURL(fnName string, operations map[string]string) (string, string, string) {
	method := ""
	route := ""
	version := ""

	subStr := strings.Split(fnName, "_")
	subStr = utils.ArrFilter(subStr, func(el string, i int) bool {
		return el != ""
	})
	j := -1

	for i, b := range subStr {

		// when set j = i
		// mean it's skip
		if j >= 0 && i < j {
			continue
		}

		s := string(b)

		// function name is not satisfied statements
		if _, ok := operations[s]; !ok && i == 0 {
			return "", "", version
		}

		if _, ok := operations[s]; ok && i == 0 {
			method = operations[s]
		}

		if s == TOKEN_VERSION {
			if i+1 < len(subStr) {
				z := i + 1
				for subStr[z] != "" {
					version += "_" + subStr[z]
					if z == len(subStr)-1 {
						break
					}
					z++
				}
			}
			version = strings.Replace(version, "_", "", 1)
			break
		}

		if _, ok := operations[s]; ok || s == TOKEN_OF {
			i++
			path := ""
			isAny := false

			for i < len(subStr) &&
				subStr[i] != TOKEN_BY &&
				subStr[i] != TOKEN_AND &&
				subStr[i] != TOKEN_OF &&
				subStr[i] != TOKEN_VERSION {

				// READ_ANY
				// or OF_ANY
				// mapped with condition line 54
				if subStr[i] == TOKEN_ANY {
					path += "*"
					isAny = true
				}

				if subStr[i] == TOKEN_FILE {
					lastWildcardIndex := strings.LastIndex(path, "*")
					if lastWildcardIndex > -1 {
						remainPath := "*"
						extension := strings.ToLower(path[lastWildcardIndex+1:])
						path = remainPath + "." + extension
					} else {
						lastWildcardIndex := strings.LastIndex(path, "_")
						if lastWildcardIndex > -1 {
							remainPath := path[:lastWildcardIndex]
							if remainPath == TOKEN_ANY {
								remainPath = "*"
							}
							extension := strings.ToLower(path[lastWildcardIndex+1:])

							path = remainPath + "." + extension
						}
					}
				}

				if subStr[i] != TOKEN_ANY && subStr[i] != TOKEN_FILE {
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
		if s == TOKEN_BY || s == TOKEN_AND {
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
		if s == TOKEN_ANY && (i == len(subStr)-1 || subStr[i+1] == TOKEN_OF) {

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

	return method, "/" + route, version
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

func ToWSEventName(n, s string) string {
	return n + "_" + utils.StrRemoveEnd(utils.StrRemoveBegin(s, "/"), "/")
}
