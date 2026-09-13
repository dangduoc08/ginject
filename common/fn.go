package common

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type singletonEntry struct {
	once sync.Once
	val  any
}

var (
	singletonsMu sync.Mutex
	singletons   = make(map[string]*singletonEntry)
)

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

		if j >= 0 && i < j {
			continue
		}

		s := b

		if i == 0 {
			var ok bool
			method, ok = HTTPOperations[s]
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

		if _, ok := HTTPOperations[s]; ok || s == TokenOf {
			i++
			path := ""
			isAny := false

			for i < len(subStr) &&
				subStr[i] != TokenBy &&
				subStr[i] != TokenAnd &&
				subStr[i] != TokenOf &&
				subStr[i] != TokenVersion {

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
			for i < len(subStr) && HTTPTokenMap[subStr[i]] == "" {
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

		if s == TokenAny && (i == len(subStr)-1 || subStr[i+1] == TokenOf) {

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
	var b strings.Builder
	b.Grow(len(rest))
	hasSeg := false
	for rest != "" {
		var p string
		p, rest, _ = strings.Cut(rest, "_")
		if p == "" {
			continue
		}
		if hasSeg {
			b.WriteByte('.')
		}
		if p == TokenAny {
			b.WriteByte('*')
		} else {
			b.WriteString(strings.ToLower(p))
		}
		hasSeg = true
	}
	if !hasSeg {
		return "", false
	}
	return b.String(), true
}

func Construct(obj any, constructor string) any {
	newObjValue := reflect.ValueOf(obj)
	key := newObjValue.Type().String()

	singletonsMu.Lock()
	entry, ok := singletons[key]
	if !ok {
		entry = &singletonEntry{}
		singletons[key] = entry
	}
	singletonsMu.Unlock()

	entry.once.Do(func() {
		if objConstructor := newObjValue.MethodByName(constructor); objConstructor.IsValid() {
			entry.val = objConstructor.Call(nil)[0].Interface()
		} else {
			entry.val = obj
		}
	})

	return entry.val
}

func ToWSEventName(s string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, "/"), "/")
}
