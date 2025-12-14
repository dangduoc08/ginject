package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/utils"
)

func kind(d any) string {
	return reflect.ValueOf(d).Kind().String()
}

func matchParams(v string) []string {
	return regexp.MustCompile(`\${(.*?)\}`).FindAllString(v, -1)
}

func isValidKey(k string) bool {
	if len(k) == 0 {
		return false
	}

	matched := regexp.MustCompile("[a-zA-Z_]+[a-zA-Z0-9_]*").FindString(k)
	return len(matched) == len(k)
}

func parseParamsToValue(k, v string, envMap map[string]any) string {
	matchedParams := matchParams(v)

	if len(matchedParams) > 0 {
		for _, param := range matchedParams {
			key := utils.StrRemoveEnd(utils.StrRemoveBegin(param, "${"), "}")
			if k == key {
				panic(fmt.Errorf("%v is assigned to itself", key))
			}
			if envMap[key] == nil {
				panic(fmt.Errorf("%v is not defined", key))
			}

			v = strings.ReplaceAll(v, param, envMap[key].(string))
		}
	}

	matchedParams = matchParams(v)
	if len(matchedParams) > 0 {
		return parseParamsToValue(k, v, envMap)
	}

	return v
}

func _flatten(org, value any, key, prefix string) {
	newPrefix := fmt.Sprintf("%v", key)
	if prefix != "" {
		newPrefix = fmt.Sprintf("%v.%v", prefix, key)
	}
	if utils.ArrIncludes([]string{"map", "slice"}, kind(value)) {
		flatten(org, value, newPrefix)
	}
	org.(map[string]any)[newPrefix] = value
}

func flatten(org, children any, prefix string) any {
	nested := org
	if prefix != "" {
		nested = children
	}

	switch nested := nested.(type) {
	case []any:
		for index, value := range nested {
			_flatten(org, value, strconv.Itoa(index), prefix)
		}
	case map[string]any:
		for key, value := range nested {
			_flatten(org, value, key, prefix)
		}
	}

	return org
}

func findRootDir(anchor string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, anchor)); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("root not found: %s", anchor)
		}
		dir = parent
	}
}
