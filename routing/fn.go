package routing

import (
	"regexp"
	"strings"

	"github.com/dangduoc08/gooh/utils"
)

func SplitRoute(str string) (string, string) {
	matchMethodReg := regexp.MustCompile(strings.Join(utils.ArrMap(HTTPMethods, func(el string, i int) string {
		return "/" + "\\" + "[" + el + "\\" + "]"
	}), "|"))

	method := matchMethodReg.FindString(str)
	noMethodRoute := matchMethodReg.ReplaceAllString(str, "")
	return method[2 : len(method)-1], noMethodRoute[:len(noMethodRoute)-1]
}

func ToEndpoint(str string) string {
	return utils.StrRemoveDup(utils.StrAddEnd(utils.StrAddBegin(utils.StrRemoveSpace(str), "/"), "/"), "/")
}

func AddMethodToRoute(str, method string) string {
	return ToEndpoint(str) + "[" + method + "]" + "/"
}

func parseToParamKey(str string) (string, map[string][]int) {
	paramKey := make(map[string][]int)

	if str != "" {
		matchParamReg := regexp.MustCompile(`\{(.*?)\}`)
		for i, s := range matchParamReg.FindAllString(str, -1) {
			str = strings.Replace(str, s, "$", 1)
			key := utils.StrRemoveEnd(utils.StrRemoveBegin(s, "{"), "}")
			paramKey[key] = append(paramKey[key], i)
		}
	}

	return str, paramKey
}

func matchWildcard(str, route string) bool {
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
func getLastWildcardNode(node *Trie, methodPattern string) *Trie {
	if node.Children["*"] != nil {
		wildcardNode := node.Children["*"]

		if wildcardNode.Children[methodPattern] != nil &&
			wildcardNode.Children[methodPattern].Index > -1 {
			return wildcardNode.Children[methodPattern]
		}
	}

	return nil
}

func isStaticRoute(route string) bool {
	return !strings.Contains(route, "*") &&
		!strings.Contains(route, "$")
}

func fromMethodtoPattern(method string) string {
	return utils.StrAddEnd(utils.StrAddBegin(method, "["), "]")
}

func fromPatternToMethod(pattern string) string {
	return utils.StrRemoveEnd(utils.StrRemoveBegin(pattern, "["), "]")
}
