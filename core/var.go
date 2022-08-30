package core

import (
	"fmt"
	"regexp"
	"strings"

	dataStructure "github.com/dangduoc08/gooh/data-structure"
)

const VAR_SYMBOL = dataStructure.DOLLAR_SIGN

type Var[T interface{}] struct{ KeyValue (map[string]T) }

func NewVar(s string) (string, *Var[interface{}]) {
	v := &Var[interface{}]{
		KeyValue: make(map[string]interface{}),
	}

	if s != "" {
		matchVarPatternReg := regexp.MustCompile(fmt.Sprintf(`\%v(.*?)\%v`, dataStructure.OPEN_CURLY_BRACKET, dataStructure.CLOSE_CURLY_BRACKET))

		for i, valB := range matchVarPatternReg.FindAll([]byte(s), -1) {
			valStr := string(valB)
			s = strings.Replace(s, valStr, VAR_SYMBOL, 1)
			varKey := dataStructure.RemoveAtEnd(dataStructure.RemoveAtBegin(valStr, dataStructure.OPEN_CURLY_BRACKET), dataStructure.CLOSE_CURLY_BRACKET)
			v.Set(varKey, i)
		}
	}

	return s, v
}

func (v *Var[T]) Get(k string) T {
	return v.KeyValue[k]
}

func (v *Var[T]) Set(k string, val T) {
	v.KeyValue[k] = val
}

func (v *Var[T]) Delete(k string) {
	delete(v.KeyValue, k)
}
