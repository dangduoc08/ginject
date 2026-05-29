package routing

import (
	"regexp"
	"strings"

	"github.com/dangduoc08/ginject/utils"
)

var matchMethodReg = regexp.MustCompile(strings.Join(utils.ArrMap(HTTPMethods, func(el string, i int) string {
	return "/" + "\\" + "[" + el + "\\" + "]"
}), "|"))

var matchParamReg = regexp.MustCompile(`\{(.*?)\}`)

func PatternToMethodRouteVersion(pattern string) (string, string, string) {
	method := matchMethodReg.FindString(pattern)
	noMethodRoute := matchMethodReg.ReplaceAllString(pattern, "")

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

func MethodRouteVersionToPattern(method, route, version string) string {
	return ToEndpoint(route) + fromVersiontoPattern(version) + "/" + fromMethodtoPattern(method) + "/"
}

func ParseToParamKey(str string) (string, map[string][]int) {
	paramKey := make(map[string][]int)

	if str == "" || strings.IndexByte(str, '{') < 0 {
		return str, paramKey
	}

	matches := matchParamReg.FindAllStringSubmatchIndex(str, -1)
	if len(matches) == 0 {
		return str, paramKey
	}

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

func matchWildcard(str, route string) bool {
	if strings.IndexByte(route, '*') < 0 {
		return str == route
	}

	subStrArr := strings.Split(route, "*")

	if len(route) < len(subStrArr) {
		return false
	}

	for i, subStr := range subStrArr {

		// s = *
		if subStr == "" {
			if i == 0 {
				nextSubStr := subStrArr[1]
				matchedIdx := strings.Index(str, nextSubStr)
				if matchedIdx < 0 {
					return false
				}
				str = str[matchedIdx:]
			} else if i == len(subStrArr)-1 {
				str = ""
			}
			continue
		} else if len(str) >= len(subStr) && str[0:len(subStr)] == subStr {
			str = str[len(subStr):]
			if i == len(subStrArr)-1 {
				continue
			}
			nextSubStr := subStrArr[i+1]
			matchedIdx := strings.Index(str, nextSubStr)
			if matchedIdx < 0 {
				return false
			}
			str = str[matchedIdx:]
			continue
		} else {
			return false
		}
	}

	return len(str) == 0
}

// get node which has * at last
func getLastWildcardNode(node *Trie, versionPattern, methodPattern string) *Trie {

	if node.Children["*"] != nil {
		wildcardNode := node.Children["*"]
		if wildcardNode.Children[versionPattern] != nil {
			wildcardNode = wildcardNode.Children[versionPattern]
		}

		if wildcardNode.Children[methodPattern] != nil &&
			wildcardNode.Children[methodPattern].Index > -1 {
			return wildcardNode.Children[methodPattern]
		}
	}

	return nil
}

func checkRouteContainsParams(route string) bool {
	return strings.Contains(route, "$")
}

func fromMethodtoPattern(method string) string {
	if method == "" {
		return method
	}
	hasL := strings.HasPrefix(method, "[")
	hasR := strings.HasSuffix(method, "]")
	if hasL && hasR {
		return method
	}
	if !hasL && !hasR {
		return "[" + method + "]"
	}
	if !hasL {
		return "[" + method
	}
	return method + "]"
}

func fromVersiontoPattern(version string) string {
	if version == "" {
		return "||"
	}
	hasL := strings.HasPrefix(version, "|")
	hasR := strings.HasSuffix(version, "|")
	if hasL && hasR {
		return version
	}
	if !hasL && !hasR {
		return "|" + version + "|"
	}
	if !hasL {
		return "|" + version
	}
	return version + "|"
}
