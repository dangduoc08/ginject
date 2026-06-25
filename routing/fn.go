package routing

import (
	"regexp"
	"strings"

	"github.com/dangduoc08/ginject/internal/slice"
)

var matchMethodReg = regexp.MustCompile(strings.Join(slice.Map(HTTPMethods, func(el string, i int) string {
	return "/" + "\\" + "[" + el + "\\" + "]"
}), "|"))

var matchParamReg = regexp.MustCompile(`\{(.*?)\}`)

var methodPatternCache = func() map[string]string {
	m := make(map[string]string, len(HTTPMethods))
	for _, method := range HTTPMethods {
		m[method] = toPattern(method, "[", "]")
	}
	return m
}()

func PatternToMethodRouteVersion(pattern string) (string, string, string) {
	method := matchMethodReg.FindString(pattern)
	noMethodRoute := pattern[:len(pattern)-len(method)]

	route := noMethodRoute[:len(noMethodRoute)-1]

	lastSlashIndex := strings.LastIndex(route, "/")
	version := ""
	if lastSlashIndex < len(route)-1 {
		version = route[lastSlashIndex+2 : len(route)-1]
	}

	route = route[:lastSlashIndex]
	method = method[2 : len(method)-1]

	return method, route, version
}

func ToEndpoint(str string) string {
	if str == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(str) + 2)
	b.WriteByte('/')
	var prev byte = '/'
	hadContent := false

	for i := 0; i < len(str); i++ {
		c := str[i]
		if isASCIISpace(c) {
			continue
		}
		hadContent = true
		if (c == '/' && prev == '/') || (c == '*' && prev == '*') {
			continue
		}
		b.WriteByte(c)
		prev = c
	}

	if !hadContent {
		return ""
	}

	if prev != '/' {
		b.WriteByte('/')
	}
	return b.String()
}

func isASCIISpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

func MethodRouteVersionToPattern(
	method,
	route,
	version string,
) string {
	routePattern := ToEndpoint(route)
	versionPattern := toPattern(version, "|", "|")
	methodPattern, ok := methodPatternCache[method]
	if !ok {
		methodPattern = toPattern(method, "[", "]")
	}

	size := len(routePattern)

	if versionPattern != "" {
		size += len(versionPattern) + 1
	}

	if methodPattern != "" {
		size += len(methodPattern) + 1
	}

	var pattern strings.Builder
	pattern.Grow(size)

	pattern.WriteString(routePattern)

	if versionPattern != "" {
		pattern.WriteString(versionPattern)
		pattern.WriteByte('/')
	}

	if methodPattern != "" {
		pattern.WriteString(methodPattern)
		pattern.WriteByte('/')
	}

	return pattern.String()
}

func ParseToParamKey(str string) (string, map[string][]int) {
	if str == "" || strings.IndexByte(str, '{') < 0 {
		return str, nil
	}

	matches := matchParamReg.FindAllStringSubmatchIndex(str, -1)
	if len(matches) == 0 {
		return str, nil
	}

	paramKey := make(map[string][]int, len(matches))
	var b strings.Builder
	b.Grow(len(str))
	prev := 0
	for i, m := range matches {
		b.WriteString(str[prev:m[0]])
		b.WriteByte('$')
		paramKey[str[m[2]:m[3]]] = append(paramKey[str[m[2]:m[3]]], i)
		prev = m[1]
	}
	b.WriteString(str[prev:])
	return b.String(), paramKey
}

func toPattern(s, l, r string) string {
	s = strings.TrimSpace(s)

	if s == "" {
		return l + r
	}

	hasL := strings.HasPrefix(s, l)
	hasR := strings.HasSuffix(s, r)
	if hasL && hasR {
		return s
	}
	if !hasL && !hasR {
		return l + s + r
	}
	if !hasL {
		return l + s
	}
	return s + r
}
