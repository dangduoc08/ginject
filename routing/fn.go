package routing

import (
	"regexp"
	"strings"

	"github.com/dangduoc08/ginject/utils"
)

func PatternToMethodRouteVersion(pattern string) (string, string, string) {
	matchMethodReg := regexp.MustCompile(strings.Join(utils.ArrMap(HTTPMethods, func(el string, i int) string {
		return "/" + "\\" + "[" + el + "\\" + "]"
	}), "|"))

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
	return utils.StrRemoveDup(
		utils.StrRemoveDup(
			utils.StrAddEnd(
				utils.StrAddBegin(
					utils.StrRemoveSpace(str), "/",
				), "/",
			),
			"/",
		),
		"*",
	)
}

func MethodRouteVersionToPattern(method, route, version string) string {
	return ToEndpoint(route) + fromVersiontoPattern(version) + "/" + fromMethodtoPattern(method) + "/"
}

func ParseToParamKey(str string) (string, map[string][]int) {
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
	return utils.StrAddEnd(utils.StrAddBegin(method, "["), "]")
}

func fromVersiontoPattern(version string) string {
	if version == "" {
		return "||"
	}
	return utils.StrAddEnd(utils.StrAddBegin(version, "|"), "|")
}
