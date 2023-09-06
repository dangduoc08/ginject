package context

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dangduoc08/gooh/utils"
)

func toJSONBuffer(args ...any) ([]byte, error) {
	data := args[0]
	switch args[0].(type) {
	case string:
		jsonStr := fmt.Sprintf(data.(string), args[1:]...)
		data = json.RawMessage(jsonStr)
	}

	return json.Marshal(&data)
}

func toJSONP(jsonStr, callback string) string {
	return fmt.Sprintf("/**/ typeof %v === 'function' && %v(%v);", callback, callback, jsonStr)
}

func getTagParams(v string) []string {
	return utils.ArrFilter[string](utils.ArrMap[string, string](
		strings.Split(v, ","), func(el string, i int) string {
			return strings.TrimSpace(el)
		}), func(el string, i int) bool {
		return el != ""
	})
}

func getTagParamIndex(v string) (int, string) {
	splittedBindParams := strings.Split(v, ".")
	bindedField := v
	bindedIndex := 0

	if len(splittedBindParams) > 1 {

		// bind:"int_5.3"
		bindedField = strings.TrimSpace(splittedBindParams[0])
		parsedInt, err := strconv.Atoi(strings.TrimSpace(splittedBindParams[1]))

		if err == nil && parsedInt > -1 {
			bindedIndex = parsedInt
		}

		return bindedIndex, bindedField
	}

	return bindedIndex, bindedField
}

// func getTagKV(arg string) (int, string) {
// 	splittedBindParams := strings.Split(v, ".")
// 	bindedField := v
// 	bindedIndex := 0

// 	if len(splittedBindParams) > 1 {

// 		// bind:"int_5.3"
// 		bindedField = strings.TrimSpace(splittedBindParams[0])
// 		parsedInt, err := strconv.Atoi(strings.TrimSpace(splittedBindParams[1]))

// 		if err == nil && parsedInt > -1 {
// 			bindedIndex = parsedInt
// 		}

// 		return bindedIndex, bindedField
// 	}

// 	return bindedIndex, bindedField
// }

func setValueToStructField(s reflect.Value) func(i int) func(v any) {
	return func(i int) func(v any) {
		return func(v any) {
			s.Elem().Field(i).Set(reflect.ValueOf(v))
		}
	}
}
