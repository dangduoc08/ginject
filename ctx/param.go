package ctx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dangduoc08/gooh/ds"
)

const PARAM_SYMBOL = ds.DOLLAR_SIGN

type Param[T interface{}] struct{ keyValue (map[string]T) }

func NewParam(str string) (string, *Param[interface{}]) {
	paramInstance := &Param[interface{}]{
		keyValue: make(map[string]interface{}),
	}

	if str != "" {
		matchParamPatternReg := regexp.MustCompile(fmt.Sprintf(`\%v(.*?)\%v`, ds.OPEN_CURLY_BRACKET, ds.CLOSE_CURLY_BRACKET))

		for paramOrder, byte := range matchParamPatternReg.FindAll([]byte(str), -1) {
			paramNameWithCurlyBracket := string(byte)
			str = strings.Replace(str, paramNameWithCurlyBracket, PARAM_SYMBOL, 1)
			paramName := ds.RemoveAtEnd(ds.RemoveAtBegin(paramNameWithCurlyBracket, ds.OPEN_CURLY_BRACKET), ds.CLOSE_CURLY_BRACKET)
			paramInstance.Set(paramName, paramOrder)
		}
	}

	return str, paramInstance
}

func (paramInstance *Param[T]) Get(key string) T {
	return paramInstance.keyValue[key]
}

func (paramInstance *Param[T]) Add(key string, value T) {
	paramInstance.keyValue[key] = value
}

func (paramInstance *Param[T]) Set(key string, value T) {
	paramInstance.keyValue[key] = value
}

func (paramInstance *Param[T]) Delete(key string) {
	delete(paramInstance.keyValue, key)
}

func (paramInstance *Param[T]) ForEach(callback func(value T, key string)) {
	for key, value := range paramInstance.keyValue {
		callback(value, key)
	}
}

func (paramInstance *Param[T]) Map(callback func(value T, key string) T) *Param[T] {
	paramInstance.ForEach(func(value T, key string) {
		paramInstance.Set(key, callback(value, key))
	})

	return paramInstance
}

func (paramInstance *Param[T]) Keys() []string {
	keyArr := make([]string, 0)
	paramInstance.ForEach(func(value T, key string) {
		keyArr = append(keyArr, key)
	})

	return keyArr
}

func (paramInstance *Param[T]) Values() []T {
	valueArr := make([]T, 0)
	paramInstance.ForEach(func(value T, key string) {
		valueArr = append(valueArr, value)
	})

	return valueArr
}
